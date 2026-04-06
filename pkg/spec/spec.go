package spec

import (
	"strconv"
	"strings"
)

// Annotation keys recognized by Augur.
const (
	AnnotationIntent     = "augur.io/intent"
	AnnotationMaxCost    = "augur.io/max-cost"
	AnnotationZones      = "augur.io/zones"
	AnnotationPriority   = "augur.io/priority"
	AnnotationReplicas   = "augur.io/replicas"
	AnnotationGPURequest = "augur.io/gpu-request"
)

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

	// GPURequest is the number of GPUs the workload needs.
	GPURequest int32 `json:"gpuRequest,omitempty"`

	// AffinityLabels are key-value pairs the workload prefers on target nodes.
	AffinityLabels map[string]string `json:"affinityLabels,omitempty"`
}

// ParseFromAnnotations extracts a WorkloadSpec from pod annotation values.
func ParseFromAnnotations(annotations map[string]string) *WorkloadSpec {
	ws := &WorkloadSpec{}

	ws.Intent = annotations[AnnotationIntent]

	if v, ok := annotations[AnnotationMaxCost]; ok {
		if cost, err := strconv.ParseFloat(v, 64); err == nil {
			ws.MaxCostPerHour = cost
		}
	}

	if v, ok := annotations[AnnotationZones]; ok {
		for _, z := range strings.Split(v, ",") {
			z = strings.TrimSpace(z)
			if z != "" {
				ws.RequiredZones = append(ws.RequiredZones, z)
			}
		}
	}

	if v, ok := annotations[AnnotationPriority]; ok {
		if p, err := strconv.ParseInt(v, 10, 32); err == nil {
			ws.Priority = int32(p)
		}
	}

	if v, ok := annotations[AnnotationReplicas]; ok {
		if r, err := strconv.ParseInt(v, 10, 32); err == nil {
			ws.Replicas = int32(r)
		}
	}

	if v, ok := annotations[AnnotationGPURequest]; ok {
		if g, err := strconv.ParseInt(v, 10, 32); err == nil {
			ws.GPURequest = int32(g)
		}
	}

	return ws
}
