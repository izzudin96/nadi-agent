//go:build linux

package collector

import "testing"

func TestParseSensorsCPUTempPackage(t *testing.T) {
	out := "coretemp-isa-0000\nAdapter: ISA adapter\nPackage id 0:  +45.0°C  (high = +80.0°C, crit = +100.0°C)\nCore 0:        +43.0°C\nCore 1:        +44.0°C\n"
	v, err := parseSensorsCPUTemp(out)
	if err != nil {
		t.Fatalf("parseSensorsCPUTemp() error = %v", err)
	}
	if v != 45.0 {
		t.Fatalf("temp = %v, want 45.0 (package)", v)
	}
}

func TestParseSensorsCPUTempFallback(t *testing.T) {
	out := "k10temp-pci-00c3\nAdapter: PCI adapter\nTctl:         +54.5°C\n"
	v, err := parseSensorsCPUTemp(out)
	if err != nil {
		t.Fatalf("parseSensorsCPUTemp() error = %v", err)
	}
	if v != 54.5 {
		t.Fatalf("temp = %v, want 54.5", v)
	}
}

func TestParseSensorsCPUTempMissing(t *testing.T) {
	if _, err := parseSensorsCPUTemp("no thermal data"); err == nil {
		t.Fatal("expected error when no temperature found")
	}
}
