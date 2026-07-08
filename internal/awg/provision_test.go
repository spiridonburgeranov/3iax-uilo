package awg

import "testing"

func TestPickInterfaceNamePrefersAwg0(t *testing.T) {
	used := map[string]struct{}{}
	if got := pickInterfaceName(57620, used); got != "awg0" {
		t.Fatalf("pickInterfaceName = %q, want awg0", got)
	}
}

func TestPickInterfaceNameUsesPortPattern(t *testing.T) {
	used := map[string]struct{}{"awg0": {}}
	if got := pickInterfaceName(57620, used); got != "awg_in_57620_ud" {
		t.Fatalf("pickInterfaceName = %q, want awg_in_57620_ud", got)
	}
}

func TestPickSubnetSkipsUsedBases(t *testing.T) {
	used := map[string]struct{}{"10.66.66.0/24": {}}
	addr, pool := pickSubnet(used)
	if pool == "10.66.66.0/24" {
		t.Fatalf("pool = %q, want next free /24", pool)
	}
	if addr == "" {
		t.Fatal("server address is empty")
	}
}

func TestParseInterfacePort(t *testing.T) {
	if got := ParseInterfacePort("awg_in_57620_ud"); got != 57620 {
		t.Fatalf("ParseInterfacePort = %d, want 57620", got)
	}
	if got := ParseInterfacePort("awg0"); got != 0 {
		t.Fatalf("ParseInterfacePort awg0 = %d, want 0", got)
	}
}
