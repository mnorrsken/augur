package spec

import (
	"context"
	"os"
	"testing"

	"github.com/mnorrsken/augur/pkg/state"
)

// policyPath returns the path to the OPA policy file relative to this package.
func policyPath(t *testing.T) string {
	t.Helper()
	// go test runs with cwd = package dir; walk up to project root.
	path := "../../config/policy.rego"
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("policy file not found at %s: %v", path, err)
	}
	return path
}

// --- Workload-level Eval tests ---

func TestEval_ValidWorkload(t *testing.T) {
	c := NewOPAClient(policyPath(t))
	ws := &WorkloadSpec{Replicas: 3, Priority: 1}

	result, err := c.Eval(context.Background(), ws)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected Allowed=true, got reasons: %v", result.Reasons)
	}
}

func TestEval_TooManyReplicas(t *testing.T) {
	c := NewOPAClient(policyPath(t))
	ws := &WorkloadSpec{Replicas: 51}

	result, err := c.Eval(context.Background(), ws)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if result.Allowed {
		t.Error("expected Allowed=false for replicas > 50")
	}
	if len(result.Reasons) == 0 {
		t.Error("expected denial reason")
	}
}

func TestEval_NegativePriority(t *testing.T) {
	c := NewOPAClient(policyPath(t))
	ws := &WorkloadSpec{Priority: -1}

	result, err := c.Eval(context.Background(), ws)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if result.Allowed {
		t.Error("expected Allowed=false for negative priority")
	}
}

func TestEval_BoundaryReplicas(t *testing.T) {
	c := NewOPAClient(policyPath(t))

	// Exactly 50 replicas should be allowed.
	ws := &WorkloadSpec{Replicas: 50}
	result, err := c.Eval(context.Background(), ws)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected Allowed=true for replicas=50, got: %v", result.Reasons)
	}
}

// --- Per-node EvalNode tests ---

func gpuNode(gpuCount float64) *state.NodeFeatures {
	return &state.NodeFeatures{
		NodeName:        "gpu-node",
		ZoneName:        "us-east-1a",
		CPUCapacity:     16,
		CPUAvailable:    8,
		MemoryCapacity:  64,
		MemoryAvailable: 32,
		GPUCount:        gpuCount,
		PodCapacity:     110,
		CostPerHour:     2.0,
		Zone:            1,
	}
}

func cheapNode(cost float64, zone string) *state.NodeFeatures {
	return &state.NodeFeatures{
		NodeName:        "node",
		ZoneName:        zone,
		CPUCapacity:     8,
		CPUAvailable:    4,
		MemoryCapacity:  16,
		MemoryAvailable: 8,
		GPUCount:        0,
		PodCapacity:     110,
		CostPerHour:     cost,
		Zone:            1,
	}
}

func TestEvalNode_GPURequestSatisfied(t *testing.T) {
	c := NewOPAClient(policyPath(t))
	ws := &WorkloadSpec{GPURequest: 2}
	node := gpuNode(4)

	result, err := c.EvalNode(context.Background(), ws, node)
	if err != nil {
		t.Fatalf("EvalNode error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected Allowed=true, got: %v", result.Reasons)
	}
}

func TestEvalNode_GPURequestNotMet(t *testing.T) {
	c := NewOPAClient(policyPath(t))
	ws := &WorkloadSpec{GPURequest: 4}
	node := gpuNode(2)

	result, err := c.EvalNode(context.Background(), ws, node)
	if err != nil {
		t.Fatalf("EvalNode error: %v", err)
	}
	if result.Allowed {
		t.Error("expected Allowed=false when node GPUs < requested")
	}
	if len(result.Reasons) == 0 {
		t.Error("expected denial reason")
	}
}

func TestEvalNode_NoGPURequestIgnoresGPUCount(t *testing.T) {
	c := NewOPAClient(policyPath(t))
	ws := &WorkloadSpec{GPURequest: 0}
	node := gpuNode(0)

	result, err := c.EvalNode(context.Background(), ws, node)
	if err != nil {
		t.Fatalf("EvalNode error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected Allowed=true when gpu_request=0, got: %v", result.Reasons)
	}
}

func TestEvalNode_CostWithinBudget(t *testing.T) {
	c := NewOPAClient(policyPath(t))
	ws := &WorkloadSpec{MaxCostPerHour: 5.0}
	node := cheapNode(3.0, "us-east-1a")

	result, err := c.EvalNode(context.Background(), ws, node)
	if err != nil {
		t.Fatalf("EvalNode error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected Allowed=true, got: %v", result.Reasons)
	}
}

func TestEvalNode_CostExceedsBudget(t *testing.T) {
	c := NewOPAClient(policyPath(t))
	ws := &WorkloadSpec{MaxCostPerHour: 2.0}
	node := cheapNode(5.0, "us-east-1a")

	result, err := c.EvalNode(context.Background(), ws, node)
	if err != nil {
		t.Fatalf("EvalNode error: %v", err)
	}
	if result.Allowed {
		t.Error("expected Allowed=false when node cost > max")
	}
}

func TestEvalNode_NoBudgetConstraint(t *testing.T) {
	c := NewOPAClient(policyPath(t))
	ws := &WorkloadSpec{MaxCostPerHour: 0} // zero means no ceiling
	node := cheapNode(999.0, "us-east-1a")

	result, err := c.EvalNode(context.Background(), ws, node)
	if err != nil {
		t.Fatalf("EvalNode error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected Allowed=true when no budget set, got: %v", result.Reasons)
	}
}

func TestEvalNode_ZoneAllowed(t *testing.T) {
	c := NewOPAClient(policyPath(t))
	ws := &WorkloadSpec{RequiredZones: []string{"us-east-1a", "us-east-1b"}}
	node := cheapNode(1.0, "us-east-1a")

	result, err := c.EvalNode(context.Background(), ws, node)
	if err != nil {
		t.Fatalf("EvalNode error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected Allowed=true for node in required zone, got: %v", result.Reasons)
	}
}

func TestEvalNode_ZoneNotAllowed(t *testing.T) {
	c := NewOPAClient(policyPath(t))
	ws := &WorkloadSpec{RequiredZones: []string{"us-east-1a"}}
	node := cheapNode(1.0, "eu-west-1a")

	result, err := c.EvalNode(context.Background(), ws, node)
	if err != nil {
		t.Fatalf("EvalNode error: %v", err)
	}
	if result.Allowed {
		t.Error("expected Allowed=false for node in wrong zone")
	}
}

func TestEvalNode_NoZoneConstraint(t *testing.T) {
	c := NewOPAClient(policyPath(t))
	ws := &WorkloadSpec{} // empty RequiredZones
	node := cheapNode(1.0, "any-zone")

	result, err := c.EvalNode(context.Background(), ws, node)
	if err != nil {
		t.Fatalf("EvalNode error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected Allowed=true with no zone constraint, got: %v", result.Reasons)
	}
}

func TestEvalNode_MultipleViolations(t *testing.T) {
	c := NewOPAClient(policyPath(t))
	ws := &WorkloadSpec{
		MaxCostPerHour: 1.0,
		GPURequest:     4,
		RequiredZones:  []string{"us-east-1a"},
	}
	node := &state.NodeFeatures{
		NodeName:    "bad-node",
		ZoneName:    "eu-west-1a", // wrong zone
		GPUCount:    0,            // not enough GPUs
		CostPerHour: 5.0,          // too expensive
		PodCapacity: 110,
	}

	result, err := c.EvalNode(context.Background(), ws, node)
	if err != nil {
		t.Fatalf("EvalNode error: %v", err)
	}
	if result.Allowed {
		t.Error("expected Allowed=false with multiple violations")
	}
	if len(result.Reasons) < 3 {
		t.Errorf("expected at least 3 denial reasons, got %d: %v", len(result.Reasons), result.Reasons)
	}
}
