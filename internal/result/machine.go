package result

import "runtime"

// MachineInfo records the hardware and OS where a benchmark was run.
// Stored in result JSON under "machine" so results are self-describing
// when shared across machines.
type MachineInfo struct {
	OS   string `json:"os"`   // e.g. "darwin", "linux"
	CPUs int    `json:"cpus"` // logical CPU count (runtime.NumCPU)
}

// DetectMachine returns a MachineInfo populated from the current host.
func DetectMachine() *MachineInfo {
	return &MachineInfo{
		OS:   runtime.GOOS,
		CPUs: runtime.NumCPU(),
	}
}
