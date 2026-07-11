package mmio

import (
	"strings"
	"testing"
)

func FuzzParseTextTrace(f *testing.F) {
	f.Add("R 4 1.000 0x1000 0x1")
	f.Add("W 1 1.000 2 0x1003 0xff")
	f.Fuzz(func(t *testing.T, input string) {
		trace, err := ParseTextTrace(strings.NewReader(input), TextTraceOptions{BARSize: 4096})
		if err == nil && len(trace.Records) == 0 {
			t.Fatal("successful parse returned no records")
		}
	})
}
