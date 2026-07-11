package barhealth

import "testing"

func TestAnalyzeClassifiesAcquisitionHealth(t *testing.T) {
	tests := []struct {
		name   string
		first  []byte
		second []byte
		want   Classification
	}{
		{"empty", nil, nil, Empty},
		{"all ff", []byte{0xff, 0xff}, nil, AllFF},
		{"all zero", []byte{0, 0}, nil, AllZero},
		{"repeating", []byte{1, 2, 1, 2}, nil, RepeatingPattern},
		{"static", []byte{1, 2, 3}, []byte{1, 2, 3}, StaticValid},
		{"dynamic", []byte{1, 2, 3}, []byte{1, 4, 3}, DynamicValid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Analyze(test.first, test.second)
			if got.Classification != test.want {
				t.Fatalf("classification = %q, want %q", got.Classification, test.want)
			}
		})
	}
}
