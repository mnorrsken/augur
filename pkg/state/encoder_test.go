package state

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEncode_Order(t *testing.T) {
	nf := NodeFeatures{
		CPUCapacity:     8,
		CPUAvailable:    4,
		MemoryCapacity:  32,
		MemoryAvailable: 16,
		GPUCount:        2,
		PodCount:        10,
		PodCapacity:     110,
		CostPerHour:     1.5,
		Zone:            2,
	}

	got := nf.Encode()
	want := []float64{8, 4, 32, 16, 2, 10, 110, 1.5, 2}

	if len(got) != len(want) {
		t.Fatalf("Encode() len = %d, want %d", len(got), len(want))
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("Encode()[%d] = %v, want %v", i, got[i], v)
		}
	}
}

func TestEncodeAll_Dimensions(t *testing.T) {
	nodes := []NodeFeatures{
		{CPUCapacity: 4},
		{CPUCapacity: 8},
		{CPUCapacity: 16},
	}

	data, rows, cols := EncodeAll(nodes)

	if rows != 3 {
		t.Errorf("rows = %d, want 3", rows)
	}
	if cols != 9 {
		t.Errorf("cols = %d, want 9", cols)
	}
	if len(data) != rows*cols {
		t.Errorf("len(data) = %d, want %d", len(data), rows*cols)
	}
	// First feature of first node.
	if data[0] != 4 {
		t.Errorf("data[0] = %v, want 4", data[0])
	}
	// First feature of second node.
	if data[9] != 8 {
		t.Errorf("data[9] = %v, want 8", data[9])
	}
}

func TestEncodeAll_Empty(t *testing.T) {
	data, rows, cols := EncodeAll(nil)
	if rows != 0 || cols != 9 || len(data) != 0 {
		t.Errorf("EncodeAll(nil) = (%v, %d, %d), want ([], 0, 9)", data, rows, cols)
	}
}

func makeNode(name string, cpuCap, memCap string, gpuCap string, podCap string, cost, zone string) v1.Node {
	capacity := v1.ResourceList{
		v1.ResourceCPU:    resource.MustParse(cpuCap),
		v1.ResourceMemory: resource.MustParse(memCap),
		v1.ResourcePods:   resource.MustParse(podCap),
	}
	allocatable := v1.ResourceList{
		v1.ResourceCPU:    resource.MustParse(cpuCap),
		v1.ResourceMemory: resource.MustParse(memCap),
	}
	if gpuCap != "" {
		capacity[v1.ResourceName(gpuResourceName)] = resource.MustParse(gpuCap)
	}
	labels := map[string]string{}
	if cost != "" {
		labels[labelCostPerHour] = cost
	}
	if zone != "" {
		labels[labelZone] = zone
	}
	return v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels},
		Status: v1.NodeStatus{
			Capacity:    capacity,
			Allocatable: allocatable,
		},
	}
}

func TestFromNodeList_Basic(t *testing.T) {
	nodes := []v1.Node{
		makeNode("node-1", "8", "16Gi", "", "110", "2.5", "us-east-1a"),
	}

	features := FromNodeList(nodes)

	if len(features) != 1 {
		t.Fatalf("len = %d, want 1", len(features))
	}
	f := features[0]

	if f.NodeName != "node-1" {
		t.Errorf("NodeName = %q, want %q", f.NodeName, "node-1")
	}
	if f.CPUCapacity != 8 {
		t.Errorf("CPUCapacity = %v, want 8", f.CPUCapacity)
	}
	// 16 GiB in bytes / (1024^3) = 16
	if f.MemoryCapacity != 16 {
		t.Errorf("MemoryCapacity = %v, want 16", f.MemoryCapacity)
	}
	if f.CostPerHour != 2.5 {
		t.Errorf("CostPerHour = %v, want 2.5", f.CostPerHour)
	}
	if f.ZoneName != "us-east-1a" {
		t.Errorf("ZoneName = %q, want %q", f.ZoneName, "us-east-1a")
	}
	if f.Zone == 0 {
		t.Errorf("Zone should be non-zero for a labelled zone")
	}
}

func TestFromNodeList_GPU(t *testing.T) {
	nodes := []v1.Node{
		makeNode("gpu-node", "16", "64Gi", "4", "110", "", ""),
	}

	features := FromNodeList(nodes)

	if features[0].GPUCount != 4 {
		t.Errorf("GPUCount = %v, want 4", features[0].GPUCount)
	}
}

func TestFromNodeList_NoGPULabel(t *testing.T) {
	nodes := []v1.Node{
		makeNode("cpu-node", "4", "8Gi", "", "110", "", ""),
	}

	features := FromNodeList(nodes)

	if features[0].GPUCount != 0 {
		t.Errorf("GPUCount = %v, want 0", features[0].GPUCount)
	}
}

func TestFromNodeList_InvalidCostLabel(t *testing.T) {
	nodes := []v1.Node{
		makeNode("node", "4", "8Gi", "", "110", "not-a-number", ""),
	}

	features := FromNodeList(nodes)

	if features[0].CostPerHour != 0 {
		t.Errorf("CostPerHour = %v, want 0 for invalid label", features[0].CostPerHour)
	}
}

func TestFromNodeList_ZoneIndexing(t *testing.T) {
	// Reset the global zone index for a predictable test.
	// Two nodes in different zones get different numeric indices.
	nodes := []v1.Node{
		makeNode("n1", "4", "8Gi", "", "110", "", "zone-a"),
		makeNode("n2", "4", "8Gi", "", "110", "", "zone-b"),
		makeNode("n3", "4", "8Gi", "", "110", "", "zone-a"),
	}

	features := FromNodeList(nodes)

	if features[0].Zone == features[1].Zone {
		t.Errorf("nodes in different zones should have different Zone indices")
	}
	if features[0].Zone != features[2].Zone {
		t.Errorf("nodes in the same zone should have the same Zone index")
	}
	if features[0].ZoneName != "zone-a" || features[1].ZoneName != "zone-b" {
		t.Errorf("ZoneName should match the label")
	}
}

func TestFromNodeList_Empty(t *testing.T) {
	features := FromNodeList(nil)
	if len(features) != 0 {
		t.Errorf("FromNodeList(nil) should return empty slice")
	}
}
