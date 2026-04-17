package core

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"

	"email-bot/config"
)

// FetchedEmail 包含单封邮件的元数据和原始 RFC-822 数据。
type FetchedEmail struct {
	UID     uint32
	Subject string
	From    string
	Date    time.Time
	Raw     []byte
}

// FetchResult 是单次获取尝试的结果。
type FetchResult struct {
	Emails     []FetchedEmail
	NewLastUID uint32 // 需要持久化的更新后高水位 UID
}

// FetchNewEmails 连接到 IMAP 服务器，获取 UID > lastUID 的所有邮件，
// 并返回邮件列表及新的高水位 UID。
//
// 首次运行行为（initialized == false）：
//   - 扫描邮箱以找到当前最大 UID。
//   - 不返回任何邮件；调用者应持久化 newLastUID 并标记源为已初始化。
//     这样可确保只转发机器人启动*之后*到达的邮件。
func FetchNewEmails(src config.SourceAccount, lastUID uint32, initialized bool) (FetchResult, error) {
	addr := fmt.Sprintf("%s:%d", src.Host, src.Port)

	c, err := client.DialTLS(addr, nil)
	if err != nil {
		return FetchResult{NewLastUID: lastUID}, fmt.Errorf("连接 %s: %w", addr, err)
	}
	defer func() { _ = c.Logout() }()

	if err := c.Login(src.Username, src.Password); err != nil {
		return FetchResult{NewLastUID: lastUID}, fmt.Errorf("登录: %w", err)
	}

	mbox, err := c.Select(src.Mailbox, true /* 只读 */)
	if err != nil {
		return FetchResult{NewLastUID: lastUID}, fmt.Errorf("选择邮箱 %q: %w", src.Mailbox, err)
	}

	if mbox.Messages == 0 {
		// 邮箱为空 — 如需要则标记为已初始化。
		return FetchResult{NewLastUID: lastUID}, nil
	}

	// ── 首次运行：找到当前最大 UID，不返回邮件 ──────────
	if !initialized {
		maxUID, err := findMaxUID(c)
		if err != nil {
			return FetchResult{NewLastUID: lastUID}, fmt.Errorf("初始化扫描: %w", err)
		}
		return FetchResult{NewLastUID: maxUID}, nil
	}

	// ── 正常运行：获取 UID > lastUID 的邮件 ─────────
	newUIDs, err := searchUIDs(c, lastUID)
	if err != nil {
		return FetchResult{NewLastUID: lastUID}, fmt.Errorf("UID 搜索: %w", err)
	}
	if len(newUIDs) == 0 {
		return FetchResult{NewLastUID: lastUID}, nil
	}

	emails, maxUID, err := fetchMessages(c, newUIDs, lastUID)
	if err != nil {
		// 返回已获取的内容。
		return FetchResult{Emails: emails, NewLastUID: maxUID}, fmt.Errorf("获取邮件: %w", err)
	}

	return FetchResult{Emails: emails, NewLastUID: maxUID}, nil
}

// findMaxUID 返回所选邮箱中当前最大的 UID。
func findMaxUID(c *client.Client) (uint32, error) {
	set := new(imap.SeqSet)
	set.AddRange(1, 0) // 1:* → 所有消息
	criteria := imap.NewSearchCriteria()
	criteria.Uid = set

	uids, err := c.UidSearch(criteria)
	if err != nil {
		return 0, err
	}
	if len(uids) == 0 {
		return 0, nil
	}
	// UID 按升序返回；最后一个元素是最大值。
	return uids[len(uids)-1], nil
}

// searchUIDs 返回所有严格大于 lastUID 的 UID。
func searchUIDs(c *client.Client, lastUID uint32) ([]uint32, error) {
	set := new(imap.SeqSet)
	set.AddRange(lastUID+1, 0) // lastUID+1:*
	criteria := imap.NewSearchCriteria()
	criteria.Uid = set
	return c.UidSearch(criteria)
}

// fetchMessages 下载 uids 中每个 UID 对应邮件的 RFC-822 正文。
func fetchMessages(c *client.Client, uids []uint32, currentMax uint32) ([]FetchedEmail, uint32, error) {
	seqset := new(imap.SeqSet)
	for _, uid := range uids {
		seqset.AddNum(uid)
	}

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{
		imap.FetchUid,
		imap.FetchEnvelope,
		section.FetchItem(),
	}

	msgCh := make(chan *imap.Message, 16)
	fetchErr := make(chan error, 1)
	go func() {
		fetchErr <- c.UidFetch(seqset, items, msgCh)
	}()

	var emails []FetchedEmail
	maxUID := currentMax

	for msg := range msgCh {
		if msg == nil {
			continue
		}
		if msg.Uid > maxUID {
			maxUID = msg.Uid
		}

		body := msg.GetBody(section)
		if body == nil {
			continue
		}
		raw, err := io.ReadAll(body)
		if err != nil {
			continue
		}

		emails = append(emails, FetchedEmail{
			UID:     msg.Uid,
			Subject: envelopeSubject(msg),
			From:    envelopeFrom(msg),
			Date:    envelopeDate(msg),
			Raw:     bytes.Clone(raw),
		})
	}

	return emails, maxUID, <-fetchErr
}

func envelopeSubject(msg *imap.Message) string {
	if msg.Envelope == nil || msg.Envelope.Subject == "" {
		return "(无主题)"
	}
	return msg.Envelope.Subject
}

func envelopeFrom(msg *imap.Message) string {
	if msg.Envelope == nil || len(msg.Envelope.From) == 0 {
		return ""
	}
	a := msg.Envelope.From[0]
	if a.PersonalName != "" {
		return fmt.Sprintf("%s <%s@%s>", a.PersonalName, a.MailboxName, a.HostName)
	}
	return fmt.Sprintf("%s@%s", a.MailboxName, a.HostName)
}

func envelopeDate(msg *imap.Message) time.Time {
	if msg.Envelope == nil {
		return time.Time{}
	}
	return msg.Envelope.Date
}
