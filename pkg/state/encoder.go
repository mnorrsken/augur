package state

import (
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// NodeFeatures holds the numeric feature vector for a single Kubernetes node.
// These features are fed to the RL agent as part of the observation space.
type NodeFeatures struct {
	// NodeName is the name of the node (not part of the feature vector).
	NodeName string
	// ZoneName is the raw availability-zone label value (used by OPA policy).
	ZoneName string
	// CPUCapacity is total CPU cores on the node.
	CPUCapacity float64
	// CPUAvailable is currently allocatable CPU cores.
	CPUAvailable float64
	// MemoryCapacity is total memory in GiB.
	MemoryCapacity float64
	// MemoryAvailable is currently allocatable memory in GiB.
	MemoryAvailable float64
	// GPUCount is the number of GPUs on the node.
	GPUCount float64
	// PodCount is the current number of pods scheduled on the node.
	PodCount float64
	// PodCapacity is the max number of pods the node can host.
	PodCapacity float64
	// CostPerHour is the hourly cost of this node (from labels or cloud API).
	CostPerHour float64
	// Zone encodes the availability zone as a numeric index.
	Zone float64
}

// Encode converts a NodeFeatures into a flat float64 slice suitable for the
// RL agent's observation tensor.
func (nf *NodeFeatures) Encode() []float64 {
	return []float64{
		nf.CPUCapacity,
		nf.CPUAvailable,
		nf.MemoryCapacity,
		nf.MemoryAvailable,
		nf.GPUCount,
		nf.PodCount,
		nf.PodCapacity,
		nf.CostPerHour,
		nf.Zone,
	}
}

// EncodeAll builds a feature matrix from a slice of NodeFeatures.
// Each row corresponds to one node. The matrix is returned as a flat slice
// in row-major order along with the dimensions.
func EncodeAll(nodes []NodeFeatures) (data []float64, rows, cols int) {
	cols = 9 // number of features per node
	rows = len(nodes)
	data = make([]float64, 0, rows*cols)
	for i := range nodes {
		data = append(data, nodes[i].Encode()...)
	}
	return data, rows, cols
}

const (
	// Well-known labels and resources.
	labelZone       = "topology.kubernetes.io/zone"
	labelCostPerHour = "node.kubernetes.io/cost-per-hour"
	gpuResourceName  = "nvidia.com/gpu"
)

// zoneIndex maps zone strings to a stable numeric index.
// Unknown zones get the next sequential index.
var zoneIndex = map[string]float64{}

func zoneToFloat(zone string) float64 {
	if zone == "" {
		return 0
	}
	if idx, ok := zoneIndex[zone]; ok {
		return idx
	}
	idx := float64(len(zoneIndex) + 1)
	zoneIndex[zone] = idx
	return idx
}

// quantityToFloat64 converts a resource.Quantity to a float64 value.
func quantityToFloat64(q resource.Quantity) float64 {
	return q.AsApproximateFloat64()
}

// bytesToGiB converts bytes to gibibytes.
func bytesToGiB(q resource.Quantity) float64 {
	return quantityToFloat64(q) / (1024 * 1024 * 1024)
}

// FromNodeList extracts NodeFeatures from real Kubernetes Node objects.
// It reads capacity/allocatable resources, GPU counts, cost labels, and zone labels.
func FromNodeList(nodes []v1.Node) []NodeFeatures {
	features := make([]NodeFeatures, 0, len(nodes))
	for i := range nodes {
		features = append(features, fromNode(&nodes[i]))
	}
	return features
}

func fromNode(node *v1.Node) NodeFeatures {
	capacity := node.Status.Capacity
	allocatable := node.Status.Allocatable
	labels := node.Labels

	cpuCap := quantityToFloat64(capacity[v1.ResourceCPU])
	cpuAlloc := quantityToFloat64(allocatable[v1.ResourceCPU])
	memCap := bytesToGiB(capacity[v1.ResourceMemory])
	memAlloc := bytesToGiB(allocatable[v1.ResourceMemory])

	var gpuCount float64
	if gpuQty, ok := capacity[v1.ResourceName(gpuResourceName)]; ok {
		gpuCount = quantityToFloat64(gpuQty)
	}

	podCap := quantityToFloat64(capacity[v1.ResourcePods])

	var costPerHour float64
	if costStr, ok := labels[labelCostPerHour]; ok {
		costStr = strings.TrimSpace(costStr)
		if v, err := strconv.ParseFloat(costStr, 64); err == nil {
			costPerHour = v
		}
	}

	zone := zoneToFloat(labels[labelZone])

	rawZone := labels[labelZone]
	return NodeFeatures{
		NodeName:        node.Name,
		ZoneName:        rawZone,
		CPUCapacity:     cpuCap,
		CPUAvailable:    cpuAlloc,
		MemoryCapacity:  memCap,
		MemoryAvailable: memAlloc,
		GPUCount:        gpuCount,
		PodCount:        0, // populated externally from pod list
		PodCapacity:     podCap,
		CostPerHour:     costPerHour,
		Zone:            zone,
	}
}
