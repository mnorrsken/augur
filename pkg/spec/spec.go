package spec

// WorkloadSpec captures the intent-based placement constraints for a workload.
// It is parsed from pod annotations or a CRD and fed into the OPA policy engine
// and the RL scoring agent.
type WorkloadSpec struct {
	// Name is the workload identifier.
	Name string `json:"name"`

	// Replicas is the desired replica count.
	Replicas int32 `json:"replicas"`

	// Priority is an integer priority (higher = more important).
	Priority int32 `json:"priority"`

	// Intent is a free-form placement intent string (e.g. "gpu-intensive", "low-latency").
	Intent string `json:"intent,omitempty"`

	// RequiredZones limits placement to specific availability zones.
	RequiredZones []string `json:"requiredZones,omitempty"`

	// MaxCostPerHour is the budget ceiling for the workload's node placement.
	MaxCostPerHour float64 `json:"maxCostPerHour,omitempty"`

	// AffinityLabels are key-value pairs the workload prefers on target nodes.
	AffinityLabels map[string]string `json:"affinityLabels,omitempty"`
}

// ParseFromAnnotations extracts a WorkloadSpec from pod annotation values.
// TODO: implement annotation parsing logic.
func ParseFromAnnotations(annotations map[string]string) (*WorkloadSpec, error) {
	// TODO: parse "augur.io/intent", "augur.io/max-cost", etc.
	return &WorkloadSpec{}, nil
}
