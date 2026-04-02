package sim

import (
	"context"
	"log"

	"github.com/mnorrsken/augur/pkg/reward"
	"github.com/mnorrsken/augur/pkg/state"
)

// Snapshot represents a point-in-time cluster state captured from Prometheus.
type Snapshot struct {
	// Timestamp is the Unix epoch seconds of the snapshot.
	Timestamp int64

	// Nodes contains the feature vectors for every node at this point in time.
	Nodes []state.NodeFeatures

	// Placements records which pod landed on which node.
	Placements []Placement
}

// Placement records a single scheduling decision and its outcome.
type Placement struct {
	PodName   string
	Namespace string
	NodeName  string
	Outcome   reward.PlacementOutcome
}

// Replayer drives offline simulation by replaying historical Prometheus snapshots
// through the RL agent and computing cumulative reward.
type Replayer struct {
	snapshots []Snapshot
}

// NewReplayer creates a Replayer from pre-loaded snapshots.
func NewReplayer(snapshots []Snapshot) *Replayer {
	return &Replayer{snapshots: snapshots}
}

// Run replays all snapshots and returns the total cumulative reward.
// TODO: call the RL agent's Score RPC for each placement and compare against
// the historical decision.
func (r *Replayer) Run(ctx context.Context) float64 {
	var totalReward float64

	for _, snap := range r.snapshots {
		for _, p := range snap.Placements {
			rw := reward.Reward(p.Outcome)
			totalReward += rw

			log.Printf("ts=%d pod=%s/%s node=%s reward=%.4f",
				snap.Timestamp, p.Namespace, p.PodName, p.NodeName, rw)
		}
	}

	// TODO: integrate with gRPC agent — send node features, receive scores,
	// compute counterfactual reward for agent-chosen nodes.

	return totalReward
}
