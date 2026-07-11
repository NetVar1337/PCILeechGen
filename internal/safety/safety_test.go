package safety

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckRejectsDefaultRouteDevice(t *testing.T) {
	sysRoot, procRoot, bdf := t.TempDir(), t.TempDir(), "0000:03:00.0"
	if err := os.MkdirAll(filepath.Join(sysRoot, "bus/pci/devices", bdf, "net", "eth0"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(procRoot, "net/route"), "Iface Destination\neth0 00000000\n")
	writeFile(t, filepath.Join(procRoot, "self/mountinfo"), "")
	if err := Check(sysRoot, procRoot, bdf); err == nil || !strings.Contains(err.Error(), "default route") {
		t.Fatalf("Check error = %v", err)
	}
}

func TestCheckRejectsRootController(t *testing.T) {
	sysRoot, procRoot, bdf := t.TempDir(), t.TempDir(), "0000:03:00.0"
	device := filepath.Join(sysRoot, "bus/pci/devices", bdf)
	if err := os.MkdirAll(device, 0o755); err != nil {
		t.Fatal(err)
	}
	blockDevice := filepath.Join(sysRoot, "devices/pci0000:00", bdf, "nvme")
	if err := os.MkdirAll(blockDevice, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(sysRoot, "dev/block/259:2"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(blockDevice, filepath.Join(sysRoot, "dev/block/259:2/device")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(procRoot, "self/mountinfo"), "1 2 259:2 / / rw - ext4 /dev/nvme0n1p2 rw\n")
	if err := Check(sysRoot, procRoot, bdf); err == nil || !strings.Contains(err.Error(), "root filesystem") {
		t.Fatalf("Check error = %v", err)
	}
}

func TestCheckFailsClosedWhenMountInfoUnavailable(t *testing.T) {
	if err := Check(t.TempDir(), t.TempDir(), "0000:03:00.0"); err == nil {
		t.Fatal("Check accepted an unclassifiable device")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
