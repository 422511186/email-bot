package core

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"email-bot/config"

	"github.com/emersion/go-message/charset"
	"golang.org/x/text/encoding/simplifiedchinese"
)

func init() {
	charset.RegisterEncoding("gbk", simplifiedchinese.GBK)
	charset.RegisterEncoding("gb18030", simplifiedchinese.GB18030)
	charset.RegisterEncoding("gb2312", simplifiedchinese.GBK)
}

func ForwardEmail(smtpCfg config.SMTPConfig, email FetchedEmail, targets []string) error {
	if len(targets) == 0 {
		return fmt.Errorf("未提供目标地址")
	}

	body := prependResentHeaders(smtpCfg.From, email.From, email.Raw)
	addr := fmt.Sprintf("%s:%d", smtpCfg.Host, smtpCfg.Port)
	auth := smtp.PlainAuth("", smtpCfg.Username, smtpCfg.Password, smtpCfg.Host)

	if smtpCfg.Port == 465 {
		return sendMailImplicitTLS(addr, smtpCfg.Host, auth, smtpCfg.From, targets, body)
	}

	return smtp.SendMail(addr, auth, smtpCfg.From, targets, body)
}

func prependResentHeaders(from, originalFrom string, original []byte) []byte {
	displayName := originalFrom
	if idx := strings.Index(originalFrom, "<"); idx >= 0 {
		displayName = strings.TrimSpace(originalFrom[idx+1:])
		if idx := strings.Index(displayName, ">"); idx >= 0 {
			displayName = displayName[:idx]
		}
	}

	header := fmt.Sprintf(
		"From: <%s>\r\nX-Forwarded-By: email-bot\r\n",
		from,
	)

	prefix := fmt.Sprintf("[%s] - ", displayName)
	modified := modifySubject(original, prefix)

	out := make([]byte, 0, len(header)+len(modified))
	out = append(out, []byte(header)...)
	out = append(out, modified...)
	return out
}

func modifySubject(original []byte, prefix string) []byte {
	s := string(original)
	lines := strings.Split(s, "\r\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "subject:") {
			lines[i] = line[:8] + prefix + strings.TrimLeft(line[8:], " \t")
			break
		}
	}
	return []byte(strings.Join(lines, "\r\n"))
}

func sendMailImplicitTLS(addr, host string, auth smtp.Auth, from string, to []string, body []byte) error {
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
	if _, err := wc.Write(body); err != nil {
		return fmt.Errorf("SMTP 写入正文: %w", err)
	}
	return wc.Close()
}


