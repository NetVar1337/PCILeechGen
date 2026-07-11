// Package conformance defines minimum driver-initialization contracts by PCI class.
package conformance

// Contract names required initialization behaviors.
type Contract struct {
	Name      string   `json:"name"`
	ClassCode uint32   `json:"class_code"`
	Behaviors []string `json:"behaviors"`
}

// ForClass returns the minimum supported initialization contract.
func ForClass(classCode uint32) Contract {
	switch {
	case classCode>>8 == 0x0108:
		return Contract{Name: "NVMe", ClassCode: classCode, Behaviors: []string{"CC.EN to CSTS.RDY", "admin queue", "identify", "features", "create IO queues", "interrupt setup"}}
	case classCode == 0x0c0330:
		return Contract{Name: "xHCI", ClassCode: classCode, Behaviors: []string{"HCRST and CNR", "RUN and HCHalted", "DCBAAP", "CRCR", "CONFIG", "command ring", "event ring", "doorbell 0", "interrupt delivery"}}
	case classCode>>16 == 0x02:
		return Contract{Name: "Ethernet", ClassCode: classCode, Behaviors: []string{"reset completion", "MAC identity", "descriptor latches", "RX/TX enable", "interrupt mask", "W1C status"}}
	case classCode>>8 == 0x0403:
		return Contract{Name: "HD Audio", ClassCode: classCode, Behaviors: []string{"controller reset", "CORB/RIRB", "stream descriptors", "interrupt status"}}
	case classCode>>8 == 0x0106:
		return Contract{Name: "SATA AHCI", ClassCode: classCode, Behaviors: []string{"host reset", "port implementation", "command list", "FIS receive", "interrupt status"}}
	default:
		return Contract{Name: "Generic", ClassCode: classCode, Behaviors: []string{"config space", "BAR decode", "reset", "interrupt setup"}}
	}
}
