package power

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWakeRequestsRuntimeOnAndReturnsD0(t *testing.T) {
	root, bdf := t.TempDir(), "0000:03:00.0"
	device := filepath.Join(root, "bus/pci/devices", bdf)
	if err := os.MkdirAll(filepath.Join(device, "power"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{"power_state": "D0", "power/runtime_status": "active", "power/control": "auto"} {
		if err := os.WriteFile(filepath.Join(device, name), []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	state, err := Wake(root, bdf, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if state.PCIState != "D0" {
		t.Fatalf("PCI state = %q", state.PCIState)
	}
	control, err := os.ReadFile(filepath.Join(device, "power/control"))
	if err != nil {
		t.Fatal(err)
	}
	if string(control) != "on" {
		t.Fatalf("power control = %q", control)
	}
}

func TestReadRejectsMissingPowerState(t *testing.T) {
	if _, err := Read(t.TempDir(), "0000:03:00.0"); err == nil {
		t.Fatal("Read accepted a missing power state")
	}
}
