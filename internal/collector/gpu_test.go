package collector

import "testing"

func TestParseNvidiaSMI(t *testing.T) {
	metrics, err := parseNvidiaSMI("42, 1024, 8192, 65, 120.5\n")
	if err != nil {
		t.Fatalf("parseNvidiaSMI() error = %v", err)
	}

	byName := map[string]float64{}
	for _, m := range metrics {
		byName[m.Name] = m.Value
	}

	if byName["gpu.usage_percent"] != 42 {
		t.Errorf("usage_percent = %v, want 42", byName["gpu.usage_percent"])
	}
	// 1024 MiB = 1073741824 bytes
	if byName["gpu.memory_used_bytes"] != 1073741824 {
		t.Errorf("memory_used_bytes = %v, want 1073741824", byName["gpu.memory_used_bytes"])
	}
	if byName["gpu.memory_total_bytes"] != 8192*1024*1024 {
		t.Errorf("memory_total_bytes = %v", byName["gpu.memory_total_bytes"])
	}
	if byName["gpu.temperature_celsius"] != 65 {
		t.Errorf("temperature_celsius = %v, want 65", byName["gpu.temperature_celsius"])
	}
	if byName["gpu.power_watts"] != 120.5 {
		t.Errorf("power_watts = %v, want 120.5", byName["gpu.power_watts"])
	}
}

func TestParseNvidiaSMIMalformed(t *testing.T) {
	if _, err := parseNvidiaSMI("42, 1024\n"); err == nil {
		t.Fatal("expected error for too-few fields")
	}
	if _, err := parseNvidiaSMI("a,b,c,d,e\n"); err == nil {
		t.Fatal("expected error for non-numeric values")
	}
}
