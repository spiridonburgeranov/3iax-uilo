package awg

import (
	"strings"
	"testing"
)

func TestNeedsObfuscationProvisionRejectsPlaceholderHeaders(t *testing.T) {
	t.Helper()
	parsed := inboundSettings{
		Jc: 4, Jmin: 64, Jmax: 256,
		S1: 15, S2: 25, S3: 35, S4: 15,
		H1: "1", H2: "2", H3: "3", H4: "4",
		I1: "<r 20>", I2: "<r 15>", I3: "<r 12>", I4: "<r 18>", I5: "<r 14>",
	}
	if !NeedsObfuscationProvision(parsed) {
		t.Fatal("expected placeholder headers to require provision")
	}
}

func TestNormalizeInboundSettingsFillsObfuscation(t *testing.T) {
	t.Helper()
	settings := `{
		"secretKey":"iJ2cBkrSGqRwIfYIDIxk7hr5RXfdR93MfJUL7yqkkH8=",
		"address":"10.66.66.1/24",
		"awgInterface":"awg0",
		"h1":"1","h2":"2","h3":"3","h4":"4",
		"jc":4,"jmin":64,"jmax":256,
		"s1":15,"s2":25,"s3":35,"s4":15,
		"clients":[],"peers":[]
	}`
	out, _, err := NormalizeInboundSettings(settings, 51820, ResourceSnapshot{})
	if err != nil {
		t.Fatalf("NormalizeInboundSettings: %v", err)
	}
	parsed, err := ParseInboundSettings(out)
	if err != nil {
		t.Fatalf("ParseInboundSettings: %v", err)
	}
	if isWeakHeader(parsed.H1) || isWeakHeader(parsed.H2) {
		t.Fatalf("headers still weak: h1=%q h2=%q", parsed.H1, parsed.H2)
	}
	if strings.TrimSpace(parsed.I1) == "" || strings.TrimSpace(parsed.I5) == "" {
		t.Fatal("expected full CPS chain")
	}
	if strings.TrimSpace(parsed.DNS) == "" {
		t.Fatal("expected default dns")
	}
}
