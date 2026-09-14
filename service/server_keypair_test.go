package service

import (
	"strings"
	"testing"
)

// The static key sing-openvpn accepts is 256 bytes of hex between its own
// markers, with anything before the begin marker ignored. TestOpenVPNStaticKey
// checks the shape; core.TestEndpointInlineTLS proves sing-box takes it.
func TestGenerateOpenVPNStaticKey(t *testing.T) {
	s := &ServerService{}
	lines := s.GenKeypair("openvpn", "")

	begin, end := -1, -1
	for index, line := range lines {
		switch line {
		case "-----BEGIN OpenVPN Static key V1-----":
			begin = index
		case "-----END OpenVPN Static key V1-----":
			end = index
		}
	}
	if begin == -1 || end <= begin {
		t.Fatalf("markers missing or out of order: %v", lines)
	}

	body := strings.Join(lines[begin+1:end], "")
	if len(body) != 512 {
		t.Errorf("key body is %d hex characters, want 512 (256 bytes)", len(body))
	}
	for _, r := range body {
		if !strings.ContainsRune("0123456789abcdef", r) {
			t.Fatalf("key body is not lowercase hex: %q", body)
		}
	}

	// Two calls must not produce the same key, or every tunnel built through
	// the panel would share one.
	if strings.Join(s.GenKeypair("openvpn", ""), "") == strings.Join(lines, "") {
		t.Error("two generated keys are identical")
	}
}
