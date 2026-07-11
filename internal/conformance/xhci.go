package conformance

const (
	xhciRunStop = uint32(1 << 0)
	xhciHCRST   = uint32(1 << 1)
	xhciHalted  = uint32(1 << 0)
	xhciCNR     = uint32(1 << 11)
)

// XHCIState models the minimum host-controller reset/run initialization transitions.
type XHCIState struct {
	USBCMD      uint32 `json:"usbcmd"`
	USBSTS      uint32 `json:"usbsts"`
	DCBAAP      uint64 `json:"dcbaap"`
	CRCR        uint64 `json:"crcr"`
	MaxSlots    uint8  `json:"max_slots"`
	ResetCycles int    `json:"reset_cycles"`
}

// NewXHCIState returns a halted controller ready for host initialization.
func NewXHCIState(resetCycles int) *XHCIState {
	if resetCycles < 1 {
		resetCycles = 1
	}
	return &XHCIState{USBSTS: xhciHalted, ResetCycles: resetCycles}
}

// WriteUSBCMD applies Run/Stop and Host Controller Reset semantics.
func (s *XHCIState) WriteUSBCMD(value uint32) {
	if value&xhciHCRST != 0 {
		s.USBCMD = xhciHCRST
		s.USBSTS = xhciHalted | xhciCNR
		return
	}
	s.USBCMD = value & xhciRunStop
	if s.USBCMD&xhciRunStop != 0 && s.DCBAAP != 0 && s.CRCR != 0 && s.MaxSlots > 0 {
		s.USBSTS &^= xhciHalted
	} else {
		s.USBSTS |= xhciHalted
	}
}

// Tick advances bounded reset completion.
func (s *XHCIState) Tick() {
	if s.USBCMD&xhciHCRST == 0 {
		return
	}
	s.ResetCycles--
	if s.ResetCycles <= 0 {
		s.USBCMD &^= xhciHCRST
		s.USBSTS &^= xhciCNR
		s.USBSTS |= xhciHalted
	}
}
