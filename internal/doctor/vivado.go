package doctor

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CheckVivado verifies executable, version, license environment, and nearby XML metadata.
func CheckVivado(path string) []Result {
	resolved, err := exec.LookPath(path)
	if err != nil {
		return []Result{{Code: "VIVADO-ENV-001", Severity: SeverityFail, Summary: "Vivado executable not found", Evidence: path}}
	}
	results := []Result{{Code: "VIVADO-ENV-001", Severity: SeverityPass, Summary: "Vivado executable found", Evidence: resolved}}
	output, err := exec.Command(resolved, "-version").CombinedOutput()
	if err != nil {
		results = append(results, Result{Code: "VIVADO-VERSION-001", Severity: SeverityFail, Summary: "Vivado version query failed", Evidence: strings.TrimSpace(string(output))})
	} else {
		results = append(results, Result{Code: "VIVADO-VERSION-001", Severity: SeverityPass, Summary: "Vivado version query succeeded", Evidence: firstLine(string(output))})
	}
	if os.Getenv("XILINXD_LICENSE_FILE") == "" && os.Getenv("LM_LICENSE_FILE") == "" {
		results = append(results, Result{Code: "VIVADO-LICENSE-002", Severity: SeverityWarn, Summary: "license environment is not set", Remediation: "preserve XILINXD_LICENSE_FILE or LM_LICENSE_FILE across sudo"})
	} else {
		results = append(results, Result{Code: "VIVADO-LICENSE-002", Severity: SeverityPass, Summary: "license environment is set"})
	}
	return append(results, checkNearbyXML(resolved)...)
}

func checkNearbyXML(executable string) []Result {
	root := filepath.Clean(filepath.Join(filepath.Dir(executable), ".."))
	matches, _ := filepath.Glob(filepath.Join(root, "data", "parts", "*.xml"))
	for _, path := range matches {
		file, err := os.Open(path)
		if err != nil {
			return []Result{{Code: "VIVADO-XML-001", Severity: SeverityFail, Summary: "cannot read Vivado part XML", Evidence: err.Error()}}
		}
		decoder := xml.NewDecoder(file)
		for {
			_, tokenErr := decoder.Token()
			if errors.Is(tokenErr, io.EOF) {
				break
			}
			if tokenErr != nil {
				file.Close()
				return []Result{{Code: "VIVADO-XML-001", Severity: SeverityFail, Summary: "invalid Vivado XML", Evidence: fmt.Sprintf("%s: %v", path, tokenErr), Remediation: "repair or reinstall the affected Vivado device files"}}
			}
		}
		file.Close()
	}
	return []Result{{Code: "VIVADO-XML-001", Severity: SeverityPass, Summary: "sampled Vivado XML is readable"}}
}

func firstLine(value string) string {
	line, _, _ := strings.Cut(value, "\n")
	return strings.TrimSpace(line)
}
