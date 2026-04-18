package core

import (
	"bytes"
	"testing"
)

func TestModifySubject_Normal(t *testing.T) {
	raw := []byte("From: a@b.com\r\nSubject: Hello\r\nTo: c@d.com\r\n\r\nBody")
	res := modifySubject(raw, "[Bot] - ")
	
	if !bytes.Contains(res, []byte("Subject: [Bot] - Hello")) {
		t.Errorf("Unexpected result: %s", res)
	}
}

func TestModifySubject_Encoded(t *testing.T) {
	// =?utf-8?B?5rWL6K+V?= means "测试"
	raw := []byte("Subject: =?utf-8?B?5rWL6K+V?=\r\n\r\nBody")
	res := modifySubject(raw, "[Prefix] ")
	
	// "[Prefix] 测试" base64 encoded -> W1ByZWZpeF0g5rWL6K+V
	expected := "Subject: =?utf-8?b?W1ByZWZpeF0g5rWL6K+V?="
	if !bytes.Contains(res, []byte(expected)) {
		t.Errorf("Unexpected result: %s", res)
	}
}

func TestModifySubject_Folded(t *testing.T) {
	// "Subject: =?utf-8?B?5rWL6K+V?=\r\n  =?utf-8?B?5rWL6K+V?=" -> "测试测试"
	raw := []byte("Subject: =?utf-8?B?5rWL6K+V?=\r\n  =?utf-8?B?5rWL6K+V?=\r\n\r\nBody")
	res := modifySubject(raw, "[P] ")
	
	// "[P] 测试测试"
	expected := "Subject: =?utf-8?b?W1BdIOa1i+ivlea1i+ivlQ==?="
	if !bytes.Contains(res, []byte(expected)) {
		t.Errorf("Unexpected result: %s", string(res))
	}
}
