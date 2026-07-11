// Package power inspects and safely requests donor D0 state.
package power

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sercanarga/pcileechgen/internal/pci"
)

// Status reports PCI and runtime power state.
type Status struct {
	PCIState      string `json:"pci_state"`
	RuntimeStatus string `json:"runtime_status"`
	Control       string `json:"control"`
}

// Read returns current donor power state.
func Read(sysRoot, bdf string) (Status, error) {
	parsed, err := pci.ParseBDF(bdf)
	if err != nil || parsed.String() != bdf {
		return Status{}, fmt.Errorf("invalid canonical PCI BDF %q", bdf)
	}
	if sysRoot == "" {
		sysRoot = "/sys"
	}
	base := filepath.Join(sysRoot, "bus/pci/devices", bdf)
	read := func(name string) string {
		data, _ := os.ReadFile(filepath.Join(base, name))
		return strings.TrimSpace(string(data))
	}
	state := Status{PCIState: read("power_state"), RuntimeStatus: read("power/runtime_status"), Control: read("power/control")}
	if state.PCIState == "" {
		return state, fmt.Errorf("power state unavailable for %s", bdf)
	}
	return state, nil
}

// Wake disables runtime autosuspend and waits for D0.
func Wake(sysRoot, bdf string, timeout time.Duration) (Status, error) {
	parsed, err := pci.ParseBDF(bdf)
	if err != nil || parsed.String() != bdf {
		return Status{}, fmt.Errorf("invalid canonical PCI BDF %q", bdf)
	}
	if sysRoot == "" {
		sysRoot = "/sys"
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	control := filepath.Join(sysRoot, "bus/pci/devices", bdf, "power/control")
	if err := os.WriteFile(control, []byte("on"), 0o200); err != nil {
		return Status{}, fmt.Errorf("disable runtime PM: %w", err)
	}
	deadline := time.Now().Add(timeout)
	for {
		state, err := Read(sysRoot, bdf)
		if err == nil && state.PCIState == "D0" {
			return state, nil
		}
		if time.Now().After(deadline) {
			return state, fmt.Errorf("device remained in %s after %s", state.PCIState, timeout)
		}
		time.Sleep(25 * time.Millisecond)
	}
}
