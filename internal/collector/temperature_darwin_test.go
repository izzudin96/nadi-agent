//go:build darwin

package collector

import "testing"

func TestParsePowermetricsCPUTemp(t *testing.T) {
	out := "**** SMC sensors ****\nCPU die temperature: 55.48 C\nGPU die temperature: 42.0 C\n"
	v, err := parsePowermetricsCPUTemp(out)
	if err != nil {
		t.Fatalf("parsePowermetricsCPUTemp() error = %v", err)
	}
	if v != 55.48 {
		t.Fatalf("temp = %v, want 55.48", v)
	}
}

func TestParsePowermetricsCPUTempMissing(t *testing.T) {
	if _, err := parsePowermetricsCPUTemp("no sensors here"); err == nil {
		t.Fatal("expected error when CPU temp line absent")
	}
}
