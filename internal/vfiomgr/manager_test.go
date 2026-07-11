package vfiomgr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareRollsBackMemberWhoseBindFails(t *testing.T) {
	root, bdf := t.TempDir(), "0000:03:00.0"
	device := filepath.Join(root, "bus/pci/devices", bdf)
	group := filepath.Join(root, "kernel/iommu_groups/17")
	native := filepath.Join(root, "bus/pci/drivers/native")
	for _, dir := range []string{device, filepath.Join(group, "devices"), native} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{filepath.Join(device, "driver_override"), filepath.Join(native, "unbind")} {
		if err := os.WriteFile(file, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(group, filepath.Join(device, "iommu_group")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(device, filepath.Join(group, "devices", bdf)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(native, filepath.Join(device, "driver")); err != nil {
		t.Fatal(err)
	}

	if _, err := New(root).Prepare(bdf, nil); err == nil {
		t.Fatal("Prepare succeeded without a vfio-pci bind endpoint")
	}
	override, err := os.ReadFile(filepath.Join(device, "driver_override"))
	if err != nil {
		t.Fatal(err)
	}
	if string(override) != "\n" {
		t.Fatalf("rollback left driver_override = %q", override)
	}
}

func TestStateSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	want := &State{Group: "17", Members: []Member{{BDF: "0000:03:00.0", OriginalDriver: "r8169"}}}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Group != want.Group || len(got.Members) != 1 || got.Members[0] != want.Members[0] {
		t.Fatalf("loaded state = %+v", got)
	}
}

func TestRestoreRejectsUntrustedStatePaths(t *testing.T) {
	tests := []Member{
		{BDF: "../../etc", OriginalDriver: "native"},
		{BDF: "0000:03:00.0", OriginalDriver: "../../evil"},
	}
	for _, member := range tests {
		err := New(t.TempDir()).Restore(&State{Members: []Member{member}})
		if err == nil {
			t.Fatalf("Restore accepted member %+v", member)
		}
	}
}
