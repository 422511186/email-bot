package core

import (
	"errors"
	"fmt"
	"strings"

	"email-bot/config"
)

// userError wraps a lower-level error and provides a short, stable message
// suitable for narrow TUI columns (the original error can still be unwrapped).
type userError struct {
	short string
	cause error
}

func (e userError) Error() string { return e.short }
func (e userError) Unwrap() error { return e.cause }

func asUserError(err error) (userError, bool) {
	var ue userError
	if errors.As(err, &ue) {
		return ue, true
	}
	return userError{}, false
}

// makeUserFacingError returns an error whose Error() is concise but actionable.
// It keeps the original error available via errors.Unwrap / errors.Is / errors.As.
func makeUserFacingError(err error, src config.SourceAccount) error {
	if err == nil {
		return nil
	}
	// Avoid double-wrapping.
	if _, ok := asUserError(err); ok {
		return err
	}

	full := err.Error()
	lower := strings.ToLower(full)

	short := ""

	// Provider security blocks (common on 163/126/yeah): server replies NO on SELECT/EXAMINE with "Unsafe Login".
	if strings.Contains(full, "Unsafe Login") || strings.Contains(lower, "kefu@188.com") {
		short = "Unsafe Login (provider blocked; enable IMAP/SMTP + use auth code/app password)"
	}

	// Generic auth failures.
	if short == "" && (strings.Contains(lower, "authentication failed") ||
		strings.Contains(lower, "invalid credentials") ||
		strings.Contains(lower, "login failed") ||
		strings.Contains(lower, "authorization failed")) {
		short = "Auth failed (use app password/auth code; regular password may be blocked)"
	}

	// Mailbox-related errors.
	if short == "" && (strings.Contains(lower, "mailbox") && (strings.Contains(lower, "doesn't exist") || strings.Contains(lower, "not found"))) {
		short = fmt.Sprintf("Mailbox not found (%s)", strings.TrimSpace(src.Mailbox))
	}

	// Timeouts.
	if short == "" && (strings.Contains(lower, "i/o timeout") || strings.Contains(lower, "timeout")) {
		short = "Timeout (check network/IMAP host/port)"
	}

	// Connection / DNS.
	if short == "" && (strings.Contains(lower, "dial tcp") || strings.Contains(lower, "no such host") || strings.Contains(lower, "connection refused")) {
		short = "Connect failed (check IMAP host/port/network)"
	}

	if short == "" {
		// Fall back to the original message; still wrap so future improvements can
		// standardize without changing call sites.
		short = full
	}

	return userError{short: short, cause: err}
}

