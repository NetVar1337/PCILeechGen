package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunClassifiesHostVFIOPowerAndBARFailures(t *testing.T) {
	root := t.TempDir()
	proc := filepath.Join(root, "proc")
	sys := filepath.Join(root, "sys")
	bdf := "0000:02:00.0"
	mustWrite(t, filepath.Join(proc, "cmdline"), "quiet splash\n")
	mustWrite(t, filepath.Join(proc, "mounts"), "overlay / overlay rw 0 0\n")
	mustWrite(t, filepath.Join(sys, "bus/pci/devices", bdf, "power_state"), "D3hot\n")
	mustWrite(t, filepath.Join(sys, "bus/pci/devices", bdf, "power/control"), "auto\n")
	mustMkdir(t, filepath.Join(sys, "bus/pci/drivers/r8169"))
	if err := os.Symlink(filepath.Join(sys, "bus/pci/drivers/r8169"), filepath.Join(sys, "bus/pci/devices", bdf, "driver")); err != nil {
		t.Fatal(err)
	}
	group := filepath.Join(sys, "kernel/iommu_groups/7/devices")
	mustMkdir(t, filepath.Join(group, bdf))
	mustMkdir(t, filepath.Join(group, "0000:02:00.1"))
	mustMkdir(t, filepath.Join(sys, "bus/pci/devices", bdf, "iommu_group"))
	if err := os.Symlink(group, filepath.Join(sys, "bus/pci/devices", bdf, "iommu_group/devices")); err != nil {
		t.Fatal(err)
	}
	mustWriteBytes(t, filepath.Join(sys, "bus/pci/devices", bdf, "resource0"), []byte{0xff, 0xff, 0xff, 0xff})

	report := Run(Options{SysRoot: sys, ProcRoot: proc, BDF: bdf})
	for _, code := range []string{"HOST-IOMMU-001", "HOST-BOOT-002", "POWER-D3-001", "VFIO-DRIVER-001", "VFIO-GROUP-001", "BAR-READ-FF-001"} {
		if !report.HasCode(code) {
			t.Errorf("missing diagnostic %s: %+v", code, report.Results)
		}
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, value string) {
	t.Helper()
	mustWriteBytes(t, path, []byte(value))
}

func mustWriteBytes(t *testing.T, path string, value []byte) {
	t.Helper()
	mustMkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, value, 0o644); err != nil {
		t.Fatal(err)
	}
}
