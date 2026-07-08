package awg

import "testing"

func TestPickInterfaceNamePrefersAwg0(t *testing.T) {
	used := map[string]struct{}{}
	if got := pickInterfaceName(used); got != "awg0" {
		t.Fatalf("pickInterfaceName = %q, want awg0", got)
	}
}

func TestPickInterfaceNameIncrements(t *testing.T) {
	used := map[string]struct{}{"awg0": {}, "awg1": {}}
	if got := pickInterfaceName(used); got != "awg2" {
		t.Fatalf("pickInterfaceName = %q, want awg2", got)
	}
}

func TestPickInterfaceNameSkipsLegacyNames(t *testing.T) {
	used := map[string]struct{}{"awg_in_57620_ud": {}}
	if got := pickInterfaceName(used); got != "awg0" {
		t.Fatalf("pickInterfaceName = %q, want awg0", got)
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

func TestParseInterfaceIndex(t *testing.T) {
	if got := ParseInterfaceIndex("awg0"); got != 0 {
		t.Fatalf("ParseInterfaceIndex awg0 = %d, want 0", got)
	}
	if got := ParseInterfaceIndex("awg3"); got != 3 {
		t.Fatalf("ParseInterfaceIndex awg3 = %d, want 3", got)
	}
	if got := ParseInterfaceIndex("awg_in_1_ud"); got != -1 {
		t.Fatalf("ParseInterfaceIndex legacy = %d, want -1", got)
	}
}

func TestBuildProvisionPlanIncludesObfuscation(t *testing.T) {
	plan, err := BuildProvisionPlan(ResourceSnapshot{
		interfaceNames: map[string]struct{}{},
		ports:          map[int]struct{}{},
		subnetBases:    map[string]struct{}{},
	})
	if err != nil {
		t.Fatal(err)
	}
	params := ObfuscationParams{
		Jc: plan.Jc, Jmin: plan.Jmin, Jmax: plan.Jmax,
		S1: plan.S1, S2: plan.S2, S3: plan.S3, S4: plan.S4,
		H1: plan.H1, H2: plan.H2, H3: plan.H3, H4: plan.H4,
		I1: plan.I1, I2: plan.I2, I3: plan.I3, I4: plan.I4, I5: plan.I5,
	}
	if err := ValidateObfuscationParams(params); err != nil {
		t.Fatalf("ValidateObfuscationParams: %v", err)
	}
	if plan.InterfaceName != "awg0" {
		t.Fatalf("InterfaceName = %q, want awg0", plan.InterfaceName)
	}
}
