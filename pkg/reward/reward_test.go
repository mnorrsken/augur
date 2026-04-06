package reward

import (
	"math"
	"testing"
	"time"
)

func TestReward_Eviction(t *testing.T) {
	o := PlacementOutcome{Evicted: true}
	got := Reward(o)
	if got != -10.0 {
		t.Errorf("Reward(evicted) = %v, want -10.0", got)
	}
}

func TestReward_NoEviction_PositiveBaseline(t *testing.T) {
	// Ideal utilization, no latency, no cost — should be near-max reward.
	o := PlacementOutcome{
		CPUUtilization:    0.6,
		MemoryUtilization: 0.6,
	}
	got := Reward(o)
	// Both util scores are 0 (exactly at 0.6), cost penalty = 0, latency = 0.
	if got != 0.0 {
		t.Errorf("Reward(perfect util, zero cost) = %v, want 0.0", got)
	}
}

func TestReward_LatencyPenalty(t *testing.T) {
	o := PlacementOutcome{
		CPUUtilization:    0.6,
		MemoryUtilization: 0.6,
		ScheduleLatency:   10 * time.Second,
		PodStartupLatency: 0,
	}
	got := Reward(o)
	// latency penalty = 10 * 0.1 = -1.0
	want := -1.0
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("Reward(10s latency) = %v, want %v", got, want)
	}
}

func TestReward_CostPenalty(t *testing.T) {
	o := PlacementOutcome{
		CPUUtilization:    0.6,
		MemoryUtilization: 0.6,
		CostPerHour:       2.0,
	}
	got := Reward(o)
	// cost penalty = 2.0 * 0.5 = -1.0
	want := -1.0
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("Reward(cost=2.0) = %v, want %v", got, want)
	}
}

func TestReward_HighUtilization(t *testing.T) {
	o := PlacementOutcome{
		CPUUtilization:    1.0,
		MemoryUtilization: 1.0,
	}
	got := Reward(o)
	// cpu_score = -(1.0 - 0.6)^2 = -0.16; mem same; total util = -1.6
	wantUtil := (-0.16 + -0.16) * 5.0 // = -1.6
	if math.Abs(got-wantUtil) > 1e-9 {
		t.Errorf("Reward(util=1.0) = %v, want %v", got, wantUtil)
	}
}

func TestReward_LowUtilization(t *testing.T) {
	o := PlacementOutcome{
		CPUUtilization:    0.0,
		MemoryUtilization: 0.0,
	}
	got := Reward(o)
	// cpu_score = -(0 - 0.6)^2 = -0.36; same for mem; total = -3.6
	wantUtil := (-0.36 + -0.36) * 5.0 // = -3.6
	if math.Abs(got-wantUtil) > 1e-9 {
		t.Errorf("Reward(util=0.0) = %v, want %v", got, wantUtil)
	}
}

// --- Intent bonus tests ---

func TestIntentBonus_GPUIntensive_Match(t *testing.T) {
	o := PlacementOutcome{Intent: "gpu-intensive", NodeHasGPU: true,
		CPUUtilization: 0.6, MemoryUtilization: 0.6}
	got := Reward(o)
	if got != 3.0 {
		t.Errorf("gpu-intensive on GPU node = %v, want 3.0", got)
	}
}

func TestIntentBonus_GPUIntensive_Mismatch(t *testing.T) {
	o := PlacementOutcome{Intent: "gpu-intensive", NodeHasGPU: false,
		CPUUtilization: 0.6, MemoryUtilization: 0.6}
	got := Reward(o)
	if got != -3.0 {
		t.Errorf("gpu-intensive on CPU node = %v, want -3.0", got)
	}
}

func TestIntentBonus_LowLatency_SameZone(t *testing.T) {
	o := PlacementOutcome{Intent: "low-latency", SameZone: true,
		CPUUtilization: 0.6, MemoryUtilization: 0.6}
	got := Reward(o)
	if got != 2.0 {
		t.Errorf("low-latency same-zone = %v, want 2.0", got)
	}
}

func TestIntentBonus_LowLatency_DifferentZone(t *testing.T) {
	o := PlacementOutcome{Intent: "low-latency", SameZone: false,
		CPUUtilization: 0.6, MemoryUtilization: 0.6}
	got := Reward(o)
	if got != -1.0 {
		t.Errorf("low-latency different-zone = %v, want -1.0", got)
	}
}

func TestIntentBonus_CostSensitive_Cheap(t *testing.T) {
	o := PlacementOutcome{Intent: "cost-sensitive", CostPerHour: 0.5,
		CPUUtilization: 0.6, MemoryUtilization: 0.6}
	got := Reward(o)
	// cost penalty: 0.5 * 0.5 = -0.25; cost-sensitive bonus: +2.0
	want := -0.25 + 2.0
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("cost-sensitive cheap node = %v, want %v", got, want)
	}
}

func TestIntentBonus_CostSensitive_Expensive(t *testing.T) {
	o := PlacementOutcome{Intent: "cost-sensitive", CostPerHour: 2.0,
		CPUUtilization: 0.6, MemoryUtilization: 0.6}
	got := Reward(o)
	// cost penalty: 2.0 * 0.5 = -1.0; no bonus
	want := -1.0
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("cost-sensitive expensive node = %v, want %v", got, want)
	}
}

func TestIntentBonus_UnknownIntent(t *testing.T) {
	o := PlacementOutcome{Intent: "batch",
		CPUUtilization: 0.6, MemoryUtilization: 0.6}
	got := Reward(o)
	if got != 0.0 {
		t.Errorf("unknown intent = %v, want 0.0", got)
	}
}

func TestIntentBonus_NoIntent(t *testing.T) {
	o := PlacementOutcome{CPUUtilization: 0.6, MemoryUtilization: 0.6}
	got := Reward(o)
	if got != 0.0 {
		t.Errorf("no intent = %v, want 0.0", got)
	}
}

func TestReward_EvictionShortCircuits(t *testing.T) {
	// Even with a great intent match, eviction should dominate.
	o := PlacementOutcome{
		Evicted:           true,
		Intent:            "gpu-intensive",
		NodeHasGPU:        true,
		CPUUtilization:    0.6,
		MemoryUtilization: 0.6,
	}
	got := Reward(o)
	if got != -10.0 {
		t.Errorf("eviction with GPU match = %v, want -10.0", got)
	}
}
