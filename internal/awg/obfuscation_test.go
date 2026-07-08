package awg

import "testing"

func TestGenerateObfuscationParamsValid(t *testing.T) {
	t.Helper()
	for range 200 {
		params := GenerateObfuscationParams()
		if err := ValidateObfuscationParams(params); err != nil {
			t.Fatalf("ValidateObfuscationParams: %v\nparams=%+v", err, params)
		}
	}
}

func TestGenerateObfuscationParamsUniquePadding(t *testing.T) {
	t.Helper()
	params := GenerateObfuscationParams()
	if !uniqueInts(params.S1, params.S2, params.S3, params.S4) {
		t.Fatalf("S values not unique: %d %d %d %d", params.S1, params.S2, params.S3, params.S4)
	}
}

func TestGenerateObfuscationParamsCPSChain(t *testing.T) {
	t.Helper()
	params := GenerateObfuscationParams()
	for _, value := range []string{params.I1, params.I2, params.I3, params.I4, params.I5} {
		if value == "" {
			t.Fatal("expected full I1-I5 CPS chain")
		}
	}
}

func TestValidateHeadersDisjointRejectsOverlap(t *testing.T) {
	t.Helper()
	err := validateHeadersDisjoint("100-200", "150-250", "300-400", "500-600")
	if err == nil {
		t.Fatal("expected overlap error")
	}
}
