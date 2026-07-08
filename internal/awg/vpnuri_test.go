package awg

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestGenerateVpnURI(t *testing.T) {
	t.Helper()
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	serverPriv, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	settings := `{
		"secretKey": "` + serverPriv + `",
		"address": "10.66.66.1/24",
		"dns": "1.1.1.1,2606:4700:4700::1111",
		"mtu": 1420,
		"jc": 4, "jmin": 64, "jmax": 256,
		"s1": 15, "s2": 25, "s3": 35, "s4": 15,
		"h1": "200000000-280000000",
		"h2": "400000000-480000000",
		"h3": "600000000-680000000",
		"h4": "350000000-430000000",
		"i1": "<r 20>", "i2": "<r 15>", "i3": "<r 12>", "i4": "<r 18>", "i5": "<r 14>"
	}`
	inbound := &model.Inbound{
		Id:       1,
		Port:     51820,
		Settings: settings,
	}
	client := ClientConfigInput{
		PrivateKey:       priv,
		PublicKey:        pub,
		AllowedIPs:       []string{"10.66.66.2/32"},
		KeepAlive:        25,
		ClientAllowedIPs: "0.0.0.0/0, ::/0",
	}
	uri, err := GenerateVpnURI(inbound, client, "203.0.113.10:51820")
	if err != nil {
		t.Fatalf("GenerateVpnURI: %v", err)
	}
	if !strings.HasPrefix(uri, "vpn://") {
		t.Fatalf("uri prefix = %q", uri[:min(8, len(uri))])
	}
	outer, err := decodeVpnURI(uri)
	if err != nil {
		t.Fatalf("decodeVpnURI: %v", err)
	}
	if outer.DefaultContainer != "amnezia-awg" {
		t.Fatalf("defaultContainer = %q", outer.DefaultContainer)
	}
	if len(outer.Containers) != 1 || outer.Containers[0].Awg.ProtocolVersion != "2" {
		t.Fatal("expected awg v2 container")
	}
	var inner vpnURIInner
	if err := json.Unmarshal([]byte(outer.Containers[0].Awg.LastConfig), &inner); err != nil {
		t.Fatalf("inner json: %v", err)
	}
	if inner.ClientIP != "10.66.66.2" || inner.Port != 51820 || inner.HostName != "203.0.113.10" {
		t.Fatalf("inner endpoint fields mismatch: %+v", inner)
	}
	if !strings.Contains(inner.Config, "Jc =") || !strings.Contains(inner.Config, "I1 =") {
		t.Fatal("embedded config missing awg fields")
	}
}

func decodeVpnURI(uri string) (vpnURIOuter, error) {
	var out vpnURIOuter
	encoded := strings.TrimPrefix(uri, "vpn://")
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return out, err
	}
	if len(raw) < 4 {
		return out, err
	}
	originalLen := binary.BigEndian.Uint32(raw[:4])
	reader, err := zlib.NewReader(bytes.NewReader(raw[4:]))
	if err != nil {
		return out, err
	}
	defer reader.Close()
	decompressed := bytes.NewBuffer(nil)
	if _, err := decompressed.ReadFrom(reader); err != nil {
		return out, err
	}
	if uint32(decompressed.Len()) != originalLen {
		return out, err
	}
	err = json.Unmarshal(decompressed.Bytes(), &out)
	return out, err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
