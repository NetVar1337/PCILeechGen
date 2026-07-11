// Package doctor diagnoses host, VFIO, donor, power, BAR, and toolchain failures.
package doctor

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sercanarga/pcileechgen/internal/pci"
)

// Severity controls doctor exit behavior and display.
type Severity string

const (
	SeverityPass Severity = "pass"
	SeverityWarn Severity = "warn"
	SeverityFail Severity = "fail"
)

// Result is one stable machine-readable diagnostic.
type Result struct {
	Code        string   `json:"code"`
	Severity    Severity `json:"severity"`
	Summary     string   `json:"summary"`
	Evidence    string   `json:"evidence,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
}

// Report is a deterministic doctor result set.
type Report struct {
	Results []Result `json:"results"`
}

// HasFailures reports whether doctor found an unsafe or blocking condition.
func (r Report) HasFailures() bool {
	for _, result := range r.Results {
		if result.Severity == SeverityFail {
			return true
		}
	}
	return false
}

// HasCode reports whether a stable diagnostic code is present.
func (r Report) HasCode(code string) bool {
	for _, result := range r.Results {
		if result.Code == code {
			return true
		}
	}
	return false
}

// Options configures host paths for real execution and fixtures.
type Options struct {
	SysRoot    string
	ProcRoot   string
	BDF        string
	VivadoPath string
}

// Run executes non-destructive reliability diagnostics.
func Run(opts Options) Report {
	if opts.SysRoot == "" {
		opts.SysRoot = "/sys"
	}
	if opts.ProcRoot == "" {
		opts.ProcRoot = "/proc"
	}
	var report Report
	report.Results = append(report.Results, checkIOMMU(opts)...)
	report.Results = append(report.Results, checkInstalledHost(opts)...)
	if opts.BDF != "" {
		report.Results = append(report.Results, checkDevice(opts)...)
	}
	if opts.VivadoPath != "" {
		report.Results = append(report.Results, CheckVivado(opts.VivadoPath)...)
	}
	sort.SliceStable(report.Results, func(i, j int) bool { return report.Results[i].Code < report.Results[j].Code })
	return report
}

func checkIOMMU(opts Options) []Result {
	cmdline, err := os.ReadFile(filepath.Join(opts.ProcRoot, "cmdline"))
	if err != nil {
		return []Result{{Code: "HOST-IOMMU-001", Severity: SeverityWarn, Summary: "cannot inspect kernel IOMMU parameters", Evidence: err.Error()}}
	}
	text := string(cmdline)
	enabled := strings.Contains(text, "intel_iommu=on") || strings.Contains(text, "amd_iommu=on")
	groups, _ := filepath.Glob(filepath.Join(opts.SysRoot, "kernel/iommu_groups", "*"))
	if !enabled || len(groups) == 0 {
		return []Result{{Code: "HOST-IOMMU-001", Severity: SeverityFail, Summary: "IOMMU is not ready", Evidence: strings.TrimSpace(text), Remediation: "enable intel_iommu=on or amd_iommu=on with iommu=pt, reboot, then verify IOMMU groups exist"}}
	}
	return []Result{{Code: "HOST-IOMMU-001", Severity: SeverityPass, Summary: "IOMMU enabled and groups present"}}
}

func checkInstalledHost(opts Options) []Result {
	mounts, err := os.ReadFile(filepath.Join(opts.ProcRoot, "mounts"))
	if err != nil {
		return nil
	}
	for _, line := range strings.Split(string(mounts), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[1] == "/" && (fields[2] == "overlay" || fields[2] == "squashfs") {
			return []Result{{Code: "HOST-BOOT-002", Severity: SeverityFail, Summary: "live or overlay root filesystem detected", Evidence: line, Remediation: "use a persistent Linux installation for VFIO work"}}
		}
	}
	return []Result{{Code: "HOST-BOOT-002", Severity: SeverityPass, Summary: "persistent root filesystem detected"}}
}

func checkDevice(opts Options) []Result {
	parsed, err := pci.ParseBDF(opts.BDF)
	if err != nil || parsed.String() != opts.BDF {
		return []Result{{Code: "PCI-DEVICE-001", Severity: SeverityFail, Summary: "invalid canonical PCI BDF", Evidence: opts.BDF}}
	}
	devicePath := filepath.Join(opts.SysRoot, "bus/pci/devices", opts.BDF)
	if _, err := os.Stat(devicePath); err != nil {
		return []Result{{Code: "PCI-DEVICE-001", Severity: SeverityFail, Summary: "PCI device not found", Evidence: opts.BDF}}
	}
	var results []Result
	results = append(results, checkPower(devicePath)...)
	results = append(results, checkDriver(devicePath)...)
	results = append(results, checkGroup(devicePath)...)
	results = append(results, checkBARs(devicePath)...)
	return results
}

func checkPower(devicePath string) []Result {
	data, err := os.ReadFile(filepath.Join(devicePath, "power_state"))
	if err != nil {
		return []Result{{Code: "POWER-D3-001", Severity: SeverityWarn, Summary: "power state unavailable", Evidence: err.Error()}}
	}
	state := strings.TrimSpace(string(data))
	if state != "D0" {
		return []Result{{Code: "POWER-D3-001", Severity: SeverityFail, Summary: "donor is not in D0", Evidence: state, Remediation: "resume the device under its native driver, set runtime PM control to on, then bind it to vfio-pci"}}
	}
	return []Result{{Code: "POWER-D3-001", Severity: SeverityPass, Summary: "donor is in D0"}}
}

func checkDriver(devicePath string) []Result {
	driver, err := filepath.EvalSymlinks(filepath.Join(devicePath, "driver"))
	if err != nil {
		return []Result{{Code: "VFIO-DRIVER-001", Severity: SeverityFail, Summary: "donor has no bound driver", Evidence: err.Error()}}
	}
	name := filepath.Base(driver)
	if name != "vfio-pci" {
		return []Result{{Code: "VFIO-DRIVER-001", Severity: SeverityFail, Summary: "donor is not bound to vfio-pci", Evidence: name, Remediation: "prepare the complete IOMMU group with pcileechgen vfio prepare"}}
	}
	return []Result{{Code: "VFIO-DRIVER-001", Severity: SeverityPass, Summary: "donor bound to vfio-pci"}}
}

func checkGroup(devicePath string) []Result {
	groupPath := filepath.Join(devicePath, "iommu_group", "devices")
	entries, err := os.ReadDir(groupPath)
	if err != nil {
		return []Result{{Code: "VFIO-GROUP-001", Severity: SeverityFail, Summary: "IOMMU group unavailable", Evidence: err.Error()}}
	}
	var mixed []string
	for _, entry := range entries {
		memberPath, resolveErr := filepath.EvalSymlinks(filepath.Join(groupPath, entry.Name()))
		if resolveErr != nil {
			memberPath = filepath.Join(groupPath, entry.Name())
		}
		driver, driverErr := filepath.EvalSymlinks(filepath.Join(memberPath, "driver"))
		if driverErr != nil || filepath.Base(driver) != "vfio-pci" {
			mixed = append(mixed, entry.Name())
		}
	}
	if len(mixed) > 0 {
		return []Result{{Code: "VFIO-GROUP-001", Severity: SeverityFail, Summary: "IOMMU group is not fully bound to vfio-pci", Evidence: strings.Join(mixed, ", "), Remediation: "bind every safe member of the group transactionally or use a physically isolated slot"}}
	}
	return []Result{{Code: "VFIO-GROUP-001", Severity: SeverityPass, Summary: "IOMMU group is viable"}}
}

func checkBARs(devicePath string) []Result {
	paths, _ := filepath.Glob(filepath.Join(devicePath, "resource[0-5]"))
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		sample := make([]byte, 4096)
		n, readErr := file.Read(sample)
		file.Close()
		if readErr != nil && n == 0 {
			continue
		}
		sample = sample[:n]
		if len(sample) == 0 {
			continue
		}
		if bytes.Count(sample, []byte{0xff}) == len(sample) {
			return []Result{{Code: "BAR-READ-FF-001", Severity: SeverityFail, Summary: "BAR sample contains only 0xFF", Evidence: filepath.Base(path), Remediation: "check D0 state, VFIO group viability, BAR mapping, and native-driver initialization requirements"}}
		}
	}
	return []Result{{Code: "BAR-READ-FF-001", Severity: SeverityPass, Summary: "no sampled BAR is uniformly 0xFF"}}
}

// Format renders a concise operator report.
func Format(report Report) string {
	var out strings.Builder
	for _, result := range report.Results {
		fmt.Fprintf(&out, "[%s] %s %s", strings.ToUpper(string(result.Severity)), result.Code, result.Summary)
		if result.Evidence != "" {
			fmt.Fprintf(&out, ": %s", result.Evidence)
		}
		out.WriteByte('\n')
		if result.Remediation != "" {
			fmt.Fprintf(&out, "       %s\n", result.Remediation)
		}
	}
	return out.String()
}
