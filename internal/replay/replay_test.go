package replay

import (
	"testing"

	"github.com/sercanarga/pcileechgen/internal/donor/mmio"
	"github.com/sercanarga/pcileechgen/internal/donor/session"
)

func TestCompareReportsFirstStructuralDivergence(t *testing.T) {
	donor := session.CaptureData{Trace: &mmio.TraceResult{Records: []mmio.AccessRecord{
		{Offset: 0x20, Type: mmio.AccessWrite, Width: 4, ByteEnable: 0xf, Value: 1, CPU: 1},
		{Offset: 0x24, Type: mmio.AccessRead, Width: 4, ByteEnable: 0xf, Value: 0},
	}}}
	emulator := session.CaptureData{Trace: &mmio.TraceResult{Records: []mmio.AccessRecord{
		{Offset: 0x20, Type: mmio.AccessWrite, Width: 4, ByteEnable: 0xf, Value: 1, CPU: 9},
		{Offset: 0x24, Type: mmio.AccessRead, Width: 4, ByteEnable: 0xf, Value: 1},
	}}}
	result := Compare(donor, emulator)
	if result.Divergence == nil || result.Divergence.Index != 1 || result.Matched != 1 {
		t.Fatalf("comparison = %+v", result)
	}
}

func TestCompareReportsLengthMismatch(t *testing.T) {
	donor := session.CaptureData{Trace: &mmio.TraceResult{Records: []mmio.AccessRecord{{Offset: 4}}}}
	result := Compare(donor, session.CaptureData{})
	if result.Divergence == nil || result.Divergence.Index != 0 {
		t.Fatalf("comparison = %+v", result)
	}
}
