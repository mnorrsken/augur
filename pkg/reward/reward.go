package reward

import "time"

// PlacementOutcome captures the observable result of a scheduling decision.
type PlacementOutcome struct {
	// NodeName is the node where the pod was placed.
	NodeName string

	// ScheduleLatency is how long the scheduling decision took.
	ScheduleLatency time.Duration

	// PodStartupLatency is the time from scheduling to the pod becoming ready.
	PodStartupLatency time.Duration

	// CPUUtilization is the node's CPU utilization after placement (0.0–1.0).
	CPUUtilization float64

	// MemoryUtilization is the node's memory utilization after placement (0.0–1.0).
	MemoryUtilization float64

	// Evicted indicates whether the pod was evicted shortly after placement.
	Evicted bool

	// CostPerHour is the effective cost of the chosen node.
	CostPerHour float64
}

// Reward computes a scalar reward signal from a placement outcome.
// The reward is designed to encourage:
//   - Low scheduling and startup latency
//   - Balanced resource utilization (not too high, not too low)
//   - No evictions
//   - Cost efficiency
//
// TODO: tune weights and non-linear transforms based on real cluster data.
func Reward(outcome PlacementOutcome) float64 {
	reward := 0.0

	// Penalize evictions heavily.
	if outcome.Evicted {
		return -10.0
	}

	// Reward low latency (normalize to seconds).
	latencyPenalty := outcome.ScheduleLatency.Seconds() + outcome.PodStartupLatency.Seconds()
	reward -= latencyPenalty * 0.1

	// Reward balanced utilization — peak reward around 0.6 utilization.
	cpuScore := -((outcome.CPUUtilization - 0.6) * (outcome.CPUUtilization - 0.6))
	memScore := -((outcome.MemoryUtilization - 0.6) * (outcome.MemoryUtilization - 0.6))
	reward += (cpuScore + memScore) * 5.0

	// Penalize high cost.
	reward -= outcome.CostPerHour * 0.5

	// TODO: add intent-match bonus, zone-preference bonus.

	return reward
}
