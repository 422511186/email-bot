package core

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"

	"email-bot/config"

	"github.com/emersion/go-message/charset"
	"golang.org/x/text/encoding/simplifiedchinese"
)

func init() {
	charset.RegisterEncoding("gbk", simplifiedchinese.GBK)
	charset.RegisterEncoding("gb18030", simplifiedchinese.GB18030)
	charset.RegisterEncoding("gb2312", simplifiedchinese.GBK)
}

type SMTPForwarder struct {
	client *smtp.Client
	from   string
}

func NewSMTPForwarder(cfg config.SMTPConfig) (*SMTPForwarder, error) {
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	dialer := &net.Dialer{Timeout: 30 * time.Second}

	var conn net.Conn
	var err error
	var client *smtp.Client

	if cfg.Port == 465 {
		tlsCfg := &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsCfg)
		if err != nil {
			return nil, fmt.Errorf("TLS 拨号: %w", err)
		}
		client, err = smtp.NewClient(conn, cfg.Host)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("SMTP 客户端: %w", err)
		}
	} else {
		conn, err = dialer.Dial("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("TCP 拨号: %w", err)
		}
		client, err = smtp.NewClient(conn, cfg.Host)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("SMTP 客户端: %w", err)
		}

		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsCfg := &tls.Config{ServerName: cfg.Host}
			if err = client.StartTLS(tlsCfg); err != nil {
				client.Close()
				return nil, fmt.Errorf("STARTTLS: %w", err)
			}
		}
	}

	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	if err := client.Auth(auth); err != nil {
		client.Close()
		return nil, fmt.Errorf("SMTP 认证: %w", err)
	}

	return &SMTPForwarder{client: client, from: cfg.From}, nil
}

func (f *SMTPForwarder) ForwardEmail(email FetchedEmail, targets []string) error {
	if len(targets) == 0 {
		return fmt.Errorf("未提供目标地址")
	}

	body := prependResentHeaders(f.from, email.From, email.Raw)

	if err := f.client.Mail(f.from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM: %w", err)
	}
	for _, rcpt := range targets {
		if err := f.client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("SMTP RCPT TO %s: %w", rcpt, err)
		}
	}

	wc, err := f.client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}

	// 无论写入成功与否，必须关闭数据流，否则会导致整个 SMTP 连接处于挂起状态而死锁
	defer func() {
		_ = wc.Close()
	}()

	if _, err := wc.Write(body); err != nil {
		return fmt.Errorf("SMTP 写入正文: %w", err)
	}

	return nil
}

func (f *SMTPForwarder) Close() error {
	if f.client != nil {
		f.client.Quit()
		return f.client.Close()
	}
	return nil
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
	headerEnd := bytes.Index(original, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		headerEnd = bytes.Index(original, []byte("\n\n"))
		if headerEnd == -1 {
			return original
		}
	}

	headers := string(original[:headerEnd])
	body := original[headerEnd:]

	// 统一处理可能混用的 \r\n 和 \n
	lines := strings.Split(strings.ReplaceAll(headers, "\r\n", "\n"), "\n")
	var outLines []string

	inSubject := false
	var subjectRaw string

	for _, line := range lines {
		lowerLine := strings.ToLower(line)
		isContinuation := strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")

		if inSubject {
			if isContinuation {
				subjectRaw += "\n" + line
				continue
			} else {
				inSubject = false
				outLines = append(outLines, processSubject(subjectRaw, prefix))
			}
		}

		if strings.HasPrefix(lowerLine, "subject:") {
			inSubject = true
			subjectRaw = line
		} else {
			outLines = append(outLines, line)
		}
	}

	if inSubject {
		outLines = append(outLines, processSubject(subjectRaw, prefix))
	}

	newHeaders := strings.Join(outLines, "\r\n")
	return append([]byte(newHeaders), body...)
}

func processSubject(rawLine, prefix string) string {
	parts := strings.SplitN(rawLine, ":", 2)
	if len(parts) != 2 {
		return rawLine
	}
	
	// 去除多行折叠带来的换行符，便于统一解码
	val := strings.ReplaceAll(parts[1], "\n", "")
	val = strings.ReplaceAll(val, "\r", "")
	val = strings.TrimSpace(val)

	dec := &mime.WordDecoder{
		CharsetReader: charset.Reader,
	}
	decoded, err := dec.DecodeHeader(val)
	if err != nil {
		// 降级处理：如果无法解码（如格式损坏），避免再次错误编码导致彻底乱码
		// 直接将其作为普通的 ascii 拼接前缀返回
		return "Subject: " + prefix + val
	}

	newSubj := prefix + decoded
	encoded := mime.BEncoding.Encode("utf-8", newSubj)

	return "Subject: " + encoded
}
