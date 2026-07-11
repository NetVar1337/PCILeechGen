// Package vfiomgr transactionally binds and restores complete IOMMU groups.
package vfiomgr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sercanarga/pcileechgen/internal/pci"
)

// Member records one group's original driver state.
type Member struct {
	BDF            string `json:"bdf"`
	OriginalDriver string `json:"original_driver,omitempty"`
}

// State is sufficient to restore a prepared group.
type State struct {
	Group   string   `json:"group"`
	Members []Member `json:"members"`
}

// Manager uses an injectable sysfs tree.
type Manager struct{ SysRoot string }

// New creates a VFIO manager.
func New(sysRoot string) *Manager {
	if sysRoot == "" {
		sysRoot = "/sys"
	}
	return &Manager{SysRoot: sysRoot}
}

// Prepare binds every member of bdf's IOMMU group to vfio-pci and rolls back on failure.
func (m *Manager) Prepare(bdf string, safe func(string) error) (_ *State, resultErr error) {
	parsed, err := pci.ParseBDF(bdf)
	if err != nil || parsed.String() != bdf {
		return nil, fmt.Errorf("invalid canonical PCI BDF %q", bdf)
	}
	groupLink := filepath.Join(m.SysRoot, "bus/pci/devices", bdf, "iommu_group")
	groupPath, err := filepath.EvalSymlinks(groupLink)
	if err != nil {
		return nil, fmt.Errorf("resolve IOMMU group: %w", err)
	}
	entries, err := os.ReadDir(filepath.Join(groupPath, "devices"))
	if err != nil {
		return nil, fmt.Errorf("list IOMMU group: %w", err)
	}
	state := &State{Group: filepath.Base(groupPath)}
	for _, entry := range entries {
		member := entry.Name()
		if safe != nil {
			if err := safe(member); err != nil {
				return nil, err
			}
		}
		driver := ""
		if path, err := filepath.EvalSymlinks(filepath.Join(m.SysRoot, "bus/pci/devices", member, "driver")); err == nil {
			driver = filepath.Base(path)
		}
		state.Members = append(state.Members, Member{BDF: member, OriginalDriver: driver})
	}
	sort.Slice(state.Members, func(i, j int) bool { return state.Members[i].BDF < state.Members[j].BDF })
	prepared := 0
	defer func() {
		if resultErr != nil {
			if restoreErr := m.restoreMembers(state.Members[:prepared]); restoreErr != nil {
				resultErr = fmt.Errorf("%w; rollback failed: %w", resultErr, restoreErr)
			}
		}
	}()
	for _, member := range state.Members {
		prepared++
		if err := m.bind(member, "vfio-pci"); err != nil {
			return nil, err
		}
	}
	return state, nil
}

// Restore returns every member to its recorded native driver.
func (m *Manager) Restore(state *State) error {
	if state == nil {
		return fmt.Errorf("VFIO state is nil")
	}
	return m.restoreMembers(state.Members)
}

func (m *Manager) restoreMembers(members []Member) error {
	var failures []string
	for _, member := range members {
		parsed, err := pci.ParseBDF(member.BDF)
		if err != nil || parsed.String() != member.BDF {
			failures = append(failures, fmt.Sprintf("invalid canonical PCI BDF %q", member.BDF))
			continue
		}
		if member.OriginalDriver == "" {
			devicePath := filepath.Join(m.SysRoot, "bus/pci/devices", member.BDF)
			if current, err := filepath.EvalSymlinks(filepath.Join(devicePath, "driver")); err == nil {
				if err := os.WriteFile(filepath.Join(current, "unbind"), []byte(member.BDF), 0o200); err != nil {
					failures = append(failures, fmt.Sprintf("restore unbound state for %s: %v", member.BDF, err))
					continue
				}
			}
			if err := os.WriteFile(filepath.Join(devicePath, "driver_override"), []byte("\n"), 0o200); err != nil {
				failures = append(failures, fmt.Sprintf("clear driver override for %s: %v", member.BDF, err))
			}
			continue
		}
		if filepath.Base(member.OriginalDriver) != member.OriginalDriver || member.OriginalDriver == "." || member.OriginalDriver == ".." {
			failures = append(failures, fmt.Sprintf("invalid original driver %q for %s", member.OriginalDriver, member.BDF))
			continue
		}
		if err := m.bind(member, member.OriginalDriver); err != nil {
			failures = append(failures, err.Error())
			continue
		}
		if err := os.WriteFile(filepath.Join(m.SysRoot, "bus/pci/devices", member.BDF, "driver_override"), []byte("\n"), 0o200); err != nil {
			failures = append(failures, fmt.Sprintf("clear driver override for %s: %v", member.BDF, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("restore VFIO group: %s", strings.Join(failures, "; "))
	}
	return nil
}

func (m *Manager) bind(member Member, driver string) error {
	devicePath := filepath.Join(m.SysRoot, "bus/pci/devices", member.BDF)
	if current, err := filepath.EvalSymlinks(filepath.Join(devicePath, "driver")); err == nil {
		if filepath.Base(current) == driver {
			return nil
		}
		if err := os.WriteFile(filepath.Join(current, "unbind"), []byte(member.BDF), 0o200); err != nil {
			return fmt.Errorf("unbind %s from %s: %w", member.BDF, filepath.Base(current), err)
		}
	}
	if err := os.WriteFile(filepath.Join(devicePath, "driver_override"), []byte(driver), 0o200); err != nil {
		return fmt.Errorf("set driver override for %s: %w", member.BDF, err)
	}
	if err := os.WriteFile(filepath.Join(m.SysRoot, "bus/pci/drivers", driver, "bind"), []byte(member.BDF), 0o200); err != nil {
		return fmt.Errorf("bind %s to %s: %w", member.BDF, driver, err)
	}
	return nil
}

// Save writes a restoration state file.
func Save(path string, state *State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

// Load reads a restoration state file.
func Load(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}
