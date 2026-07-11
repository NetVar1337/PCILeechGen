// Package replay compares donor and emulator initialization sessions.
package replay

import (
	"fmt"

	"github.com/sercanarga/pcileechgen/internal/donor/mmio"
	"github.com/sercanarga/pcileechgen/internal/donor/session"
)

// Divergence records the first behavior mismatch.
type Divergence struct {
	Index    int                `json:"index"`
	Reason   string             `json:"reason"`
	Donor    *mmio.AccessRecord `json:"donor,omitempty"`
	Emulator *mmio.AccessRecord `json:"emulator,omitempty"`
}

// Result summarizes normalized initialization replay.
type Result struct {
	Matched    int         `json:"matched"`
	DonorTotal int         `json:"donor_total"`
	EmuTotal   int         `json:"emulator_total"`
	Divergence *Divergence `json:"divergence,omitempty"`
}

// Compare returns the first structural MMIO divergence, ignoring timestamps and CPU IDs.
func Compare(donor, emulator session.CaptureData) Result {
	result := Result{}
	if donor.Trace != nil {
		result.DonorTotal = len(donor.Trace.Records)
	}
	if emulator.Trace != nil {
		result.EmuTotal = len(emulator.Trace.Records)
	}
	limit := result.DonorTotal
	if result.EmuTotal < limit {
		limit = result.EmuTotal
	}
	for i := range limit {
		a, b := donor.Trace.Records[i], emulator.Trace.Records[i]
		if a.Offset != b.Offset || a.Type != b.Type || a.Width != b.Width || a.ByteEnable != b.ByteEnable || a.Value != b.Value {
			reason := fmt.Sprintf("MMIO mismatch at record %d", i)
			result.Divergence = &Divergence{Index: i, Reason: reason, Donor: &a, Emulator: &b}
			return result
		}
		result.Matched++
	}
	if result.DonorTotal != result.EmuTotal {
		result.Divergence = &Divergence{Index: limit, Reason: "trace lengths differ"}
	}
	return result
}

// OK reports complete normalized agreement.
func (r Result) OK() bool { return r.Divergence == nil }
