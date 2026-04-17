package core

import (
	"crypto/tls"
	"fmt"
	"io"
	"strings"
	"time"

	"email-bot/internal/config"
	"email-bot/internal/state"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"golang.org/x/text/encoding/simplifiedchinese"
	"gopkg.in/gomail.v2"
)

func init() {
	// Register Chinese charsets to handle legacy email encodings
	charset.RegisterEncoding("gbk", simplifiedchinese.GBK)
	charset.RegisterEncoding("gb18030", simplifiedchinese.GB18030)
	charset.RegisterEncoding("gb2312", simplifiedchinese.GBK) // GB2312 can be handled by GBK
}

type Processor struct {
	cfg     *config.Config
	state   *state.Manager
	targets map[string]config.Target
	logger  *LogManager
}

func NewProcessor(cfg *config.Config, st *state.Manager, logger *LogManager) *Processor {
	targets := make(map[string]config.Target)
	for _, t := range cfg.Targets {
		targets[t.ID] = t
	}

	return &Processor{
		cfg:     cfg,
		state:   st,
		targets: targets,
		logger:  logger,
	}
}

func (p *Processor) Start() {
	p.logger.Info("Starting email processor daemon...")

	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()

	// Initial run
	p.runCycle()

	for range ticker.C {
		p.runCycle()
	}
}

func (p *Processor) runCycle() {
	for _, source := range p.cfg.Sources {
		// Find targets for this source based on rules
		var targetIDs []string
		for _, rule := range p.cfg.Rules {
			for _, sID := range rule.Sources {
				if sID == source.ID {
					targetIDs = append(targetIDs, rule.Targets...)
				}
			}
		}

		// Remove duplicate targets
		targetMap := make(map[string]bool)
		var uniqueTargets []string
		for _, tID := range targetIDs {
			if !targetMap[tID] {
				targetMap[tID] = true
				uniqueTargets = append(uniqueTargets, tID)
			}
		}

		if len(uniqueTargets) == 0 {
			continue // No forwarding rules for this source
		}

		go p.processSource(source, uniqueTargets)
	}
}

func (p *Processor) processSource(source config.Source, targetIDs []string) {
	p.logger.Infof("[%s] Connecting to %s...", source.ID, source.Host)

	c, err := client.DialTLS(source.Host, nil)
	if err != nil {
		p.logger.Errorf("[%s] Failed to connect: %v", source.ID, err)
		return
	}
	defer c.Logout()

	if err := c.Login(source.Username, source.Password); err != nil {
		p.logger.Errorf("[%s] Login failed: %v", source.ID, err)
		return
	}

	mbox, err := c.Select("INBOX", false)
	if err != nil {
		p.logger.Errorf("[%s] Select INBOX failed: %v", source.ID, err)
		return
	}

	if mbox.Messages == 0 {
		p.logger.Infof("[%s] No messages in INBOX", source.ID)
		return
	}

	lastUID := p.state.GetUID(source.ID)
	p.logger.Infof("[%s] Last processed UID: %d, Total messages: %d", source.ID, lastUID, mbox.Messages)

	// Build search criteria to fetch new messages
	seqset := new(imap.SeqSet)
	if lastUID == 0 {
		// If no state, just fetch the latest 10 messages or less
		start := uint32(1)
		if mbox.Messages > 10 {
			start = mbox.Messages - 9
		}
		seqset.AddRange(start, mbox.Messages)
	} else {
		// We use UID search
		criteria := imap.NewSearchCriteria()
		criteria.Uid = new(imap.SeqSet)
		criteria.Uid.AddRange(lastUID+1, 0)
		
		uids, err := c.UidSearch(criteria)
		if err != nil {
			p.logger.Errorf("[%s] UID search failed: %v", source.ID, err)
			return
		}
		
		if len(uids) == 0 {
			p.logger.Infof("[%s] No new messages.", source.ID)
			return
		}
		
		// If the only UID returned is the lastUID itself (due to how * works when there's no newer message)
		if len(uids) == 1 && uids[0] <= lastUID {
			p.logger.Infof("[%s] No new messages.", source.ID)
			return
		}

		for _, uid := range uids {
			if uid > lastUID {
				seqset.AddNum(uid)
			}
		}
	}

	if seqset.Empty() {
		return
	}

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	// Fetch ENVELOPE, BODYSTRUCTURE, and BODY.PEEK[]
	section := &imap.BodySectionName{Peek: true}
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid, section.FetchItem()}

	go func() {
		if lastUID == 0 {
			done <- c.Fetch(seqset, items, messages)
		} else {
			done <- c.UidFetch(seqset, items, messages)
		}
	}()

	var maxUID uint32 = lastUID

	for msg := range messages {
		if msg.Uid > maxUID {
			maxUID = msg.Uid
		}

		body := msg.GetBody(section)
		if body == nil {
			p.logger.Errorf("[%s] Failed to get message body for UID %d", source.ID, msg.Uid)
			continue
		}

		// Parse the message
		m, err := mail.CreateReader(body)
		if err != nil {
			p.logger.Errorf("[%s] Failed to parse message UID %d: %v", source.ID, msg.Uid, err)
			continue
		}

		header := m.Header
		subject, _ := header.Subject()
		from, _ := header.AddressList("From")
		
		fromStr := "Unknown"
		if len(from) > 0 {
			fromStr = from[0].String()
		}

		p.logger.Infof("[%s] Processing message UID %d: %s (From: %s)", source.ID, msg.Uid, subject, fromStr)

		// Read parts
		var textBody, htmlBody string
		
		for {
			part, err := m.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				p.logger.Errorf("[%s] Error reading part for UID %d: %v", source.ID, msg.Uid, err)
				break
			}

			switch h := part.Header.(type) {
			case *mail.InlineHeader:
				b, _ := io.ReadAll(part.Body)
				contentType, _, _ := h.ContentType()
				if strings.HasPrefix(contentType, "text/html") {
					htmlBody = string(b)
				} else if strings.HasPrefix(contentType, "text/plain") {
					textBody = string(b)
				}
			case *mail.AttachmentHeader:
				// Skip downloading attachments to save memory, or handle small ones
				// In a full implementation, we'd buffer to temp files
				p.logger.Infof("[%s] Skipping attachment in UID %d to save bandwidth", source.ID, msg.Uid)
			}
		}

		// Forward to targets
		for _, tID := range targetIDs {
			target, ok := p.targets[tID]
			if !ok {
				continue
			}

			err := p.forwardEmail(target, source.ID, subject, fromStr, textBody, htmlBody)
			if err != nil {
				p.logger.Errorf("[%s->%s] Failed to forward UID %d: %v", source.ID, tID, msg.Uid, err)
			} else {
				p.logger.Infof("[%s->%s] Successfully forwarded UID %d", source.ID, tID, msg.Uid)
			}
		}
	}

	if err := <-done; err != nil {
		p.logger.Errorf("[%s] Fetch error: %v", source.ID, err)
	} else if maxUID > lastUID {
		// Update state
		if err := p.state.SetUID(source.ID, maxUID); err != nil {
			p.logger.Errorf("[%s] Failed to save state: %v", source.ID, err)
		} else {
			p.logger.Infof("[%s] Updated last processed UID to %d", source.ID, maxUID)
		}
	}
}

func (p *Processor) forwardEmail(target config.Target, sourceID, subject, fromStr, textBody, htmlBody string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", target.Email)
	m.SetHeader("To", target.Email) // Forward to the target's own email address
	m.SetHeader("Subject", fmt.Sprintf("Fwd: [%s] %s", sourceID, subject))

	// Construct forwarding body
	headerInfo := fmt.Sprintf("----- Forwarded Message -----\nFrom: %s\nSubject: %s\n\n", fromStr, subject)

	if htmlBody != "" {
		htmlHeaderInfo := strings.ReplaceAll(headerInfo, "\n", "<br>")
		m.SetBody("text/html", htmlHeaderInfo+htmlBody)
		if textBody != "" {
			m.AddAlternative("text/plain", headerInfo+textBody)
		}
	} else if textBody != "" {
		m.SetBody("text/plain", headerInfo+textBody)
	} else {
		m.SetBody("text/plain", headerInfo+"(No content)")
	}

	// Extract SMTP host and port
	hostParts := strings.Split(target.Host, ":")
	host := hostParts[0]
	port := 465
	if len(hostParts) > 1 {
		fmt.Sscanf(hostParts[1], "%d", &port)
	}

	d := gomail.NewDialer(host, port, target.Username, target.Password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true} // Simplify for testing

	return d.DialAndSend(m)
}
