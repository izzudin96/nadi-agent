package collector

import (
	"testing"
)

func TestFirstDevice(t *testing.T) {
	scan := "/dev/sda -d sat # /dev/sda, ATA device\n/dev/sdb -d sat # /dev/sdb, ATA device\n"
	if got := firstDevice(scan); got != "/dev/sda" {
		t.Fatalf("firstDevice() = %q, want /dev/sda", got)
	}
	if got := firstDevice("no devices here"); got != "" {
		t.Fatalf("firstDevice() = %q, want empty", got)
	}
}

func TestParseSmartctlJSON(t *testing.T) {
	data := []byte(`{
		"smart_status": {"passed": true},
		"ata_smart_attributes": {
			"table": [
				{"id": 5,  "name": "Reallocated_Sector_Ct",    "raw": {"value": 2}},
				{"id": 9,  "name": "Power_On_Hours",           "raw": {"value": 8765}},
				{"id": 194,"name": "Temperature_Celsius",      "raw": {"value": 42}},
				{"id": 197,"name": "Current_Pending_Sector",   "raw": {"value": 0}},
				{"id": 198,"name": "Offline_Uncorrectable",    "raw": {"value": 0}}
			]
		}
	}`)

	metrics, err := parseSmartctlJSON(data)
	if err != nil {
		t.Fatalf("parseSmartctlJSON() error = %v", err)
	}

	byName := map[string]float64{}
	for _, m := range metrics {
		byName[m.Name] = m.Value
	}

	if byName["disk.smart.health_passed"] != 1 {
		t.Errorf("health_passed = %v, want 1", byName["disk.smart.health_passed"])
	}
	if byName["disk.smart.reallocated_sectors"] != 2 {
		t.Errorf("reallocated_sectors = %v, want 2", byName["disk.smart.reallocated_sectors"])
	}
	if byName["disk.smart.power_on_hours"] != 8765 {
		t.Errorf("power_on_hours = %v, want 8765", byName["disk.smart.power_on_hours"])
	}
	if byName["disk.smart.temperature_celsius"] != 42 {
		t.Errorf("temperature_celsius = %v, want 42", byName["disk.smart.temperature_celsius"])
	}
}

func TestParseSmartctlJSONHealthFailed(t *testing.T) {
	metrics, err := parseSmartctlJSON([]byte(`{"smart_status":{"passed":false}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(metrics) != 1 || metrics[0].Value != 0 {
		t.Fatalf("expected single health_passed=0, got %v", metrics)
	}
}

func TestParseSmartctlJSONInvalid(t *testing.T) {
	if _, err := parseSmartctlJSON([]byte("not json")); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
