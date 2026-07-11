package conformance

import "testing"

func TestXHCIStateResetAndRunSequence(t *testing.T) {
	state := NewXHCIState(2)
	if state.USBSTS&xhciHalted == 0 {
		t.Fatal("controller must reset halted")
	}
	state.WriteUSBCMD(xhciHCRST)
	state.Tick()
	if state.USBCMD&xhciHCRST == 0 || state.USBSTS&xhciCNR == 0 {
		t.Fatal("reset cleared before configured delay")
	}
	state.Tick()
	if state.USBCMD&xhciHCRST != 0 || state.USBSTS&xhciCNR != 0 || state.USBSTS&xhciHalted == 0 {
		t.Fatal("reset did not complete into halted state")
	}
	state.DCBAAP, state.CRCR, state.MaxSlots = 0x1000, 0x2000, 8
	state.WriteUSBCMD(xhciRunStop)
	if state.USBSTS&xhciHalted != 0 {
		t.Fatal("configured controller did not enter running state")
	}
}

func TestContractsCoverSupportedInitializationClasses(t *testing.T) {
	for _, class := range []uint32{0x010802, 0x0c0330, 0x020000, 0x040300, 0x010601} {
		contract := ForClass(class)
		if contract.Name == "Generic" || len(contract.Behaviors) == 0 {
			t.Fatalf("class 0x%06x has no specific contract", class)
		}
	}
}
