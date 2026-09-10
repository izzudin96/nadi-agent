//go:build darwin

package collector

import "testing"

func TestParsePowermetricsIntel(t *testing.T) {
	out := "*** Sampled system activity ***\nIntel energy model derived package power (CPUs+GT+SA): 12.34W\n"
	v, err := parsePowermetricsPackagePower(out)
	if err != nil {
		t.Fatalf("parsePowermetricsPackagePower() error = %v", err)
	}
	if v != 12.34 {
		t.Fatalf("watts = %v, want 12.34", v)
	}
}

func TestParsePowermetricsAppleSilicon(t *testing.T) {
	out := "Combined Power (CPU + GPU + ANE): 234.5 mW\n"
	v, err := parsePowermetricsPackagePower(out)
	if err != nil {
		t.Fatalf("parsePowermetricsPackagePower() error = %v", err)
	}
	if v != 0.2345 {
		t.Fatalf("watts = %v, want 0.2345 (mW->W)", v)
	}
}

func TestParsePowermetricsMissing(t *testing.T) {
	if _, err := parsePowermetricsPackagePower("no power data"); err == nil {
		t.Fatal("expected error when no power line present")
	}
}
