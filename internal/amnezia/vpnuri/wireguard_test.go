package vpnuri_test

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/amnezia/vpnuri"
)

func TestFromWireGuardConfig(t *testing.T) {
	t.Helper()
	conf := `[Interface]
PrivateKey = aP7kZx9vQ2mN8rT5wY1uJ4hF6gD0sL3eB9cX2nM5pR8tW1yU4iO7aS0dF3gH6jK9l=
Address = 10.0.0.2/32
DNS = 1.1.1.1, 1.0.0.1
MTU = 1280

[Peer]
PublicKey = bQ8lAy0wR3nO9sU6xZ2vK5iG7hE1tM4fC0dY3oN6qS9uX2zV5jP8bT1eG4iJ7kM0n=
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint = 203.0.113.10:51820
PersistentKeepalive = 25`
	uri, err := vpnuri.FromWireGuardConfig(conf, "WireGuard test")
	if err != nil {
		t.Fatalf("FromWireGuardConfig: %v", err)
	}
	if !strings.HasPrefix(uri, "vpn://") {
		t.Fatalf("uri prefix = %q", uri[:8])
	}
}

func TestFromXrayConfig(t *testing.T) {
	t.Helper()
	config := []byte(`{
		"remarks": "vless test",
		"outbounds": [{
			"protocol": "vless",
			"settings": {
				"vnext": [{
					"address": "203.0.113.20",
					"port": 443,
					"users": [{"id": "00000000-0000-0000-0000-000000000001"}]
				}]
			}
		}]
	}`)
	uri, err := vpnuri.FromXrayConfig(config, "vless test", "", "amnezia-xray")
	if err != nil {
		t.Fatalf("FromXrayConfig: %v", err)
	}
	if !strings.HasPrefix(uri, "vpn://") {
		t.Fatalf("uri prefix = %q", uri[:8])
	}
	var outer map[string]any
	if err := json.Unmarshal(mustDecodeVpnURI(t, uri), &outer); err != nil {
		t.Fatalf("decode outer: %v", err)
	}
	if outer["defaultContainer"] != "amnezia-xray" {
		t.Fatalf("defaultContainer = %v", outer["defaultContainer"])
	}
	if outer["hostName"] != "203.0.113.20" {
		t.Fatalf("hostName = %v", outer["hostName"])
	}
}

func mustDecodeVpnURI(t *testing.T, uri string) []byte {
	t.Helper()
	raw := strings.TrimPrefix(uri, "vpn://")
	decoded, err := decodeAmneziaPayload(raw)
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	return decoded
}

func decodeAmneziaPayload(encoded string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	if len(raw) < 4 {
		return nil, err
	}
	reader, err := zlib.NewReader(bytes.NewReader(raw[4:]))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	decompressed := bytes.NewBuffer(nil)
	if _, err := decompressed.ReadFrom(reader); err != nil {
		return nil, err
	}
	originalLen := binary.BigEndian.Uint32(raw[:4])
	if uint32(decompressed.Len()) != originalLen {
		return nil, err
	}
	return decompressed.Bytes(), nil
}
