// Package barhealth classifies BAR acquisition without conflating mapping and data failures.
package barhealth

import "bytes"

// Classification describes sampled BAR contents.
type Classification string

const (
	Empty            Classification = "empty"
	AllFF            Classification = "all-ff"
	AllZero          Classification = "all-zero"
	RepeatingPattern Classification = "repeating-pattern"
	StaticValid      Classification = "static-valid"
	DynamicValid     Classification = "dynamic-valid"
)

// Result summarizes one or more BAR samples.
type Result struct {
	Classification Classification `json:"classification"`
	Size           int            `json:"size"`
	ChangedBytes   int            `json:"changed_bytes"`
}

// Analyze classifies one snapshot and an optional repeated snapshot.
func Analyze(first, second []byte) Result {
	result := Result{Size: len(first)}
	if len(first) == 0 {
		result.Classification = Empty
		return result
	}
	if bytes.Count(first, []byte{0xff}) == len(first) {
		result.Classification = AllFF
		return result
	}
	if bytes.Count(first, []byte{0x00}) == len(first) {
		result.Classification = AllZero
		return result
	}
	if repeating(first) {
		result.Classification = RepeatingPattern
		return result
	}
	for i := 0; i < len(first) && i < len(second); i++ {
		if first[i] != second[i] {
			result.ChangedBytes++
		}
	}
	if result.ChangedBytes > 0 {
		result.Classification = DynamicValid
	} else {
		result.Classification = StaticValid
	}
	return result
}

func repeating(data []byte) bool {
	for width := 1; width <= 16 && width*2 <= len(data); width++ {
		ok := true
		for i := width; i < len(data); i++ {
			if data[i] != data[i%width] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}
