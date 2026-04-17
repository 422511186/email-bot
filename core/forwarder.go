package core

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/smtp"
	"strings"
	"time"

	"email-bot/config"

	"github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"golang.org/x/text/encoding/simplifiedchinese"
)

func init() {
	charset.RegisterEncoding("gbk", simplifiedchinese.GBK)
	charset.RegisterEncoding("gb18030", simplifiedchinese.GB18030)
	charset.RegisterEncoding("gb2312", simplifiedchinese.GBK)
}

// ForwardEmail 转发邮件，保留完整 MIME 结构（含内联图片和附件）。
func ForwardEmail(smtpCfg config.SMTPConfig, email FetchedEmail, targets []string) error {
	if len(targets) == 0 {
		return fmt.Errorf("未提供目标地址")
	}

	m := buildMessage(smtpCfg, email, targets)
	addr := fmt.Sprintf("%s:%d", smtpCfg.Host, smtpCfg.Port)

	if smtpCfg.Port == 465 {
		return sendImplicitTLS(addr, smtpCfg.Host, smtpCfg.Username, smtpCfg.Password, smtpCfg.From, targets, m)
	}

	return sendSTARTTLS(addr, smtpCfg.Username, smtpCfg.Password, smtpCfg.From, targets, m)
}

type messagePart struct {
	contentType      string
	body             []byte
	transferEncoding string
	contentID        string
	filename         string
	isAttachment     bool
}

type message struct {
	from         string
	to           []string
	subject      string
	date         time.Time
	parts        []messagePart
	originalFrom string
}

func buildMessage(smtpCfg config.SMTPConfig, email FetchedEmail, targets []string) *message {
	m := &message{
		from:    smtpCfg.From,
		to:      targets,
		subject: "Fwd: " + email.Subject,
		date:    time.Now(),
	}

	reader, err := mail.NewReader(bytes.NewReader(email.Raw))
	if err != nil {
		m.parts = append(m.parts, messagePart{
			contentType: "text/plain; charset=utf-8",
			body:        []byte("(无法解析原始邮件内容)"),
		})
		return m
	}

	// 提取原始邮件的 From 头部（用于 X-Original-From）
	m.originalFrom = reader.Header.Get("From")

	for {
		p, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		body, _ := io.ReadAll(p.Body)
		var part messagePart

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			ct, params, _ := h.ContentType()
			part.contentType = ct
			part.body = body
			part.contentID = h.Get("Content-Id")
			part.transferEncoding = h.Get("Content-Transfer-Encoding")
			part.isAttachment = strings.Contains(h.Get("Content-Disposition"), "attachment")
			if fn, ok := params["name"]; ok {
				part.filename = fn
			}
		case *mail.AttachmentHeader:
			ct, params, _ := h.ContentType()
			part.contentType = ct
			part.body = body
			part.isAttachment = true
			part.filename = h.Get("Filename")
			if fn, ok := params["name"]; ok {
				part.filename = fn
			}
		default:
			part.contentType = "application/octet-stream"
			part.body = body
		}

		m.parts = append(m.parts, part)
	}

	return m
}

func (m *message) WriteTo(w io.Writer) (int64, error) {
	var buf bytes.Buffer

	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString(fmt.Sprintf("From: %s\r\n", m.from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(m.to, ", ")))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", encodeHeader(m.subject)))
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", m.date.Format(time.RFC1123Z)))
	buf.WriteString(fmt.Sprintf("X-Forwarded-By: email-bot\r\n"))
	if m.originalFrom != "" {
		buf.WriteString(fmt.Sprintf("X-Original-From: %s\r\n", m.originalFrom))
	}

	if len(m.parts) == 0 {
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n(无正文内容)\r\n")
		return buf.WriteTo(w)
	}

	if len(m.parts) == 1 && strings.HasPrefix(m.parts[0].contentType, "text/") && !m.parts[0].isAttachment {
		p := m.parts[0]
		buf.WriteString(fmt.Sprintf("Content-Type: %s\r\n", p.contentType))
		buf.WriteString("\r\n")
		buf.Write(p.body)
		return buf.WriteTo(w)
	}

	boundary := fmt.Sprintf("----=_NextPart_%d", time.Now().UnixNano()%1000000)
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
	buf.WriteString("\r\n")

	var textParts []messagePart
	var otherParts []messagePart

	for _, p := range m.parts {
		if strings.HasPrefix(p.contentType, "text/plain") || strings.HasPrefix(p.contentType, "text/html") {
			if !p.isAttachment {
				textParts = append(textParts, p)
				continue
			}
		}
		otherParts = append(otherParts, p)
	}

	if len(textParts) > 0 {
		if len(textParts) > 1 {
			altBoundary := fmt.Sprintf("----=_AltPart_%d", time.Now().UnixNano()%1000000)
			buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", altBoundary))
			buf.WriteString("\r\n")

			for _, p := range textParts {
				buf.WriteString(fmt.Sprintf("--%s\r\n", altBoundary))
				buf.WriteString(fmt.Sprintf("Content-Type: %s\r\n", p.contentType))
				buf.WriteString("\r\n")
				buf.Write(p.body)
				buf.WriteString("\r\n")
			}
			buf.WriteString(fmt.Sprintf("--%s--\r\n\r\n", altBoundary))
		} else {
			p := textParts[0]
			buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			buf.WriteString(fmt.Sprintf("Content-Type: %s\r\n", p.contentType))
			buf.WriteString("\r\n")
			buf.Write(p.body)
			buf.WriteString("\r\n")
		}
	}

	for _, p := range otherParts {
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString(fmt.Sprintf("Content-Type: %s", p.contentType))
		if p.filename != "" {
			buf.WriteString(fmt.Sprintf("; name=\"%s\"", p.filename))
		}
		buf.WriteString("\r\n")

		if p.contentID != "" {
			buf.WriteString(fmt.Sprintf("Content-ID: %s\r\n", p.contentID))
		}
		if p.isAttachment {
			buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", p.filename))
		} else if p.contentID != "" {
			buf.WriteString("Content-Disposition: inline\r\n")
		}
		buf.WriteString("Content-Transfer-Encoding: base64\r\n")
		buf.WriteString("\r\n")

		encoder := base64.NewEncoder(base64.StdEncoding, &buf)
		encoder.Write(p.body)
		encoder.Close()
		buf.WriteString("\r\n")
	}

	buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	return buf.WriteTo(w)
}

// encodeHeader encodes a header value using RFC 2047 if it contains non-ASCII characters.
func encodeHeader(s string) string {
	for _, r := range s {
		if r > 127 {
			return fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(s)))
		}
	}
	return s
}

func sendImplicitTLS(addr, host, username, password, from string, to []string, m *message) error {
	tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("TLS 拨号: %w", err)
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP 客户端: %w", err)
	}
	defer func() { _ = c.Quit() }()

	auth := smtp.PlainAuth("", username, password, host)
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("SMTP 认证: %w", err)
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM: %w", err)
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("SMTP RCPT TO %s: %w", rcpt, err)
		}
	}

	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err := m.WriteTo(wc); err != nil {
		return fmt.Errorf("SMTP 写入正文: %w", err)
	}
	return wc.Close()
}

func sendSTARTTLS(addr, username, password, from string, to []string, m *message) error {
	auth := smtp.PlainAuth("", username, password, strings.Split(addr, ":")[0])
	return smtp.SendMail(addr, auth, from, to, func(w io.Writer) error {
		_, err := m.WriteTo(w)
		return err
	})
}
