package svgen

import (
	"strings"
	"testing"
)

func TestGeneratedXHCIResetsHaltedAndHostOwned(t *testing.T) {
	generated, err := GenerateBarImplDeviceSV(xhciConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, contract := range []string{
		"reg_0x00000020 <= 32'h00000000",
		"reg_0x00000024 <= 32'h00000001",
		"reg_0x00000020[1] <= 1'b0",
		"reg_0x00000024[0] <= 1'b0",
	} {
		if !strings.Contains(generated, contract) {
			t.Fatalf("generated xHCI HDL missing contract %q", contract)
		}
	}
}
