package state

// NodeFeatures holds the numeric feature vector for a single Kubernetes node.
// These features are fed to the RL agent as part of the observation space.
type NodeFeatures struct {
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

// TODO: add FromNodeList(nodes []v1.Node) []NodeFeatures that extracts
// features from real Kubernetes Node objects using resource.Quantity parsing.
