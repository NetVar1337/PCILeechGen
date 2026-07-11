package pci

import "testing"

func FuzzCapabilityParsing(f *testing.F) {
	f.Add([]byte{0xff, 0xff, 0xff, 0xff})
	f.Add(make([]byte, ConfigSpaceLegacySize))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > ConfigSpaceSize {
			input = input[:ConfigSpaceSize]
		}
		cs := NewConfigSpaceFromBytes(input)
		_ = ParseCapabilities(cs)
		_ = ParseExtCapabilities(cs)
		_ = ValidateCapabilityChains(cs)
	})
}
