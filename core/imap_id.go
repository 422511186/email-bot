package core

import (
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

func isNeteaseIMAPHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	// NetEase consumer mail domains commonly used for IMAP.
	switch h {
	case "imap.163.com", "imap.126.com", "imap.yeah.net":
		return true
	default:
		return false
	}
}

// trySendClientID sends the RFC 2971 IMAP ID command when supported/needed.
// Some providers (notably NetEase 163/126) may block third-party IMAP access
// unless the client identifies itself.
func trySendClientID(c *client.Client, host string) {
	if c == nil {
		return
	}

	should := false
	if ok, err := c.Support("ID"); err == nil && ok {
		should = true
	}
	if !should && isNeteaseIMAPHost(host) {
		// Attempt even if capability is not advertised for some reason.
		should = true
	}
	if !should {
		return
	}

	// Per RFC 2971: ID NIL or ID ("key" "value" ...).
	// Keep it short and ASCII.
	cmd := &imap.Command{
		Name: "ID",
		Arguments: []interface{}{
			[]interface{}{
				"name", "email-bot",
				"vendor", "email-bot",
			},
		},
	}

	if status, err := c.Execute(cmd, nil); err == nil && status != nil {
		// If the server rejects ID, don't fail the whole fetch; the subsequent
		// SELECT/EXAMINE will still provide a clear error.
		_ = status.Err()
	}
}

