// Package safety prevents disruptive operations on host-critical PCI devices.
package safety

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sercanarga/pcileechgen/internal/pci"
)

// Check rejects devices serving the default route or root filesystem ancestry.
func Check(sysRoot, procRoot, bdf string) error {
	parsed, err := pci.ParseBDF(bdf)
	if err != nil || parsed.String() != bdf {
		return fmt.Errorf("invalid canonical PCI BDF %q", bdf)
	}
	if sysRoot == "" {
		sysRoot = "/sys"
	}
	if procRoot == "" {
		procRoot = "/proc"
	}
	devicePath := filepath.Join(sysRoot, "bus/pci/devices", bdf)
	interfaces, _ := os.ReadDir(filepath.Join(devicePath, "net"))
	for _, iface := range interfaces {
		active, routeErr := defaultRoute(filepath.Join(procRoot, "net/route"), iface.Name())
		if routeErr != nil {
			return fmt.Errorf("cannot classify default-route safety for %s: %w", bdf, routeErr)
		}
		if active {
			return fmt.Errorf("%s owns the active default route through %s", bdf, iface.Name())
		}
	}
	mountInfo, err := os.Open(filepath.Join(procRoot, "self/mountinfo"))
	if err != nil {
		return fmt.Errorf("cannot classify root-filesystem safety for %s: %w", bdf, err)
	}
	defer mountInfo.Close()
	scanner := bufio.NewScanner(mountInfo)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 6 || fields[4] != "/" {
			continue
		}
		majorMinor := fields[2]
		blockPath, resolveErr := filepath.EvalSymlinks(filepath.Join(sysRoot, "dev/block", majorMinor, "device"))
		if resolveErr == nil && strings.Contains(blockPath, bdf) {
			return fmt.Errorf("%s contains the active root filesystem controller", bdf)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("cannot classify root-filesystem safety for %s: %w", bdf, err)
	}
	return nil
}

func defaultRoute(path, iface string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == iface && fields[1] == "00000000" {
			return true, nil
		}
	}
	return false, scanner.Err()
}
