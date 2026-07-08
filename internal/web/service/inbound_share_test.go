package service

import (
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestResolveShareEndpointRejectsAwgInterfaceHost(t *testing.T) {
	t.Helper()
	svc := InboundService{}
	inbound := &model.Inbound{
		Port:              51821,
		Listen:            "203.0.113.5",
		ShareAddrStrategy: "listen",
	}
	got := svc.ResolveShareEndpoint(inbound, "awg", "awg:51821")
	want := "203.0.113.5:51821"
	if got != want {
		t.Fatalf("endpoint = %q, want %q", got, want)
	}
}

func TestAdvertisableEndpointHostRejectsInterfaceNames(t *testing.T) {
	t.Helper()
	for _, host := range []string{"awg", "awg1", "wg0", "inbound-awg2"} {
		if advertisableEndpointHost(host) != "" {
			t.Fatalf("advertisableEndpointHost(%q) should be empty", host)
		}
	}
}

func TestAdvertisableEndpointHostAcceptsServerIP(t *testing.T) {
	t.Helper()
	if got := advertisableEndpointHost("203.0.113.10:51821"); got != "203.0.113.10" {
		t.Fatalf("advertisableEndpointHost = %q, want 203.0.113.10", got)
	}
}

func TestJoinShareHostPortIPv6(t *testing.T) {
	t.Helper()
	got := joinShareHostPort("2001:db8::1", 51820)
	if got != "[2001:db8::1]:51820" {
		t.Fatalf("joinShareHostPort = %q", got)
	}
}
