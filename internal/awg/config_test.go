package awg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfigFileAmneziaWGv2(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "awg0.conf")
	content := strings.Join([]string{
		"[Interface]",
		"PrivateKey = server-priv-key=",
		"Address = 10.66.66.1/24",
		"ListenPort = 51820",
		"MTU = 1420",
		"DNS = 1.1.1.1",
		"Jc = 4",
		"Jmin = 64",
		"Jmax = 256",
		"S1 = 15",
		"S2 = 25",
		"S3 = 35",
		"S4 = 15",
		"H1 = 123",
		"H2 = 456",
		"H3 = 789",
		"H4 = 1011",
		"I1 = <b 0x01>",
		"",
		"[Peer]",
		"# alice@example.com",
		"PublicKey = peer-pub-key=",
		"PresharedKey = peer-psk=",
		"AllowedIPs = 10.66.66.2/32",
		"PersistentKeepalive = 25",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	parsed, err := ParseConfigFile(path)
	if err != nil {
		t.Fatalf("ParseConfigFile: %v", err)
	}
	if parsed.ListenPort != 51820 {
		t.Fatalf("ListenPort = %d, want 51820", parsed.ListenPort)
	}
	if parsed.Jc != 4 || parsed.S3 != 35 || parsed.H2 != "456" || parsed.I1 != "<b 0x01>" {
		t.Fatalf("obfuscation not parsed: %+v", parsed)
	}
	if len(parsed.Peers) != 1 {
		t.Fatalf("peers = %d, want 1", len(parsed.Peers))
	}
	if parsed.Peers[0].Name != "alice@example.com" {
		t.Fatalf("peer name = %q", parsed.Peers[0].Name)
	}
	if parsed.Peers[0].PublicKey != "peer-pub-key=" {
		t.Fatalf("peer public key = %q", parsed.Peers[0].PublicKey)
	}
}
