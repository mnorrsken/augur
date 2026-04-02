"""Custom Gym environment wrapping a simulated Kubernetes cluster.

The agent observes node feature vectors and selects which node to place
the next pending pod on. Reward is computed via the reward function in
pkg/reward.
"""

import gymnasium as gym
import numpy as np
from gymnasium import spaces


class KubeSchedulerEnv(gym.Env):
    """Simulated Kubernetes scheduling environment for RL training.

    Observation: a flattened feature matrix of shape (num_nodes * features_per_node,).
    Action: discrete — index of the node to place the pod on.
    Reward: scalar reward based on placement outcome.
    """

    metadata = {"render_modes": []}

    FEATURES_PER_NODE = 9  # must match state.NodeFeatures field count

    def __init__(self, num_nodes: int = 20) -> None:
        super().__init__()
        self.num_nodes = num_nodes

        self.observation_space = spaces.Box(
            low=0.0,
            high=np.inf,
            shape=(num_nodes * self.FEATURES_PER_NODE,),
            dtype=np.float32,
        )
        self.action_space = spaces.Discrete(num_nodes)

        self._state: np.ndarray | None = None
        self._step_count = 0
        self._max_steps = 200

    def reset(self, *, seed: int | None = None, options: dict | None = None):
        super().reset(seed=seed)
        self._step_count = 0
        self._state = self._generate_cluster_state()
        return self._state, {}

    def step(self, action: int):
        assert self.action_space.contains(action), f"invalid action {action}"
        self._step_count += 1

        # TODO: simulate actual pod placement on the chosen node.
        # For now, compute a placeholder reward.
        chosen_node_features = self._get_node_features(action)
        reward = self._compute_reward(chosen_node_features)

        # Update cluster state after placement.
        # TODO: reduce available resources on the chosen node.

        terminated = False
        truncated = self._step_count >= self._max_steps
        self._state = self._generate_cluster_state()

        return self._state, reward, terminated, truncated, {}

    def _generate_cluster_state(self) -> np.ndarray:
        """Generate a random cluster state for simulation.

        TODO: replace with replay of real Prometheus snapshots or a
        higher-fidelity cluster model.
        """
        rng = self.np_random
        state = np.zeros(self.num_nodes * self.FEATURES_PER_NODE, dtype=np.float32)

        for i in range(self.num_nodes):
            base = i * self.FEATURES_PER_NODE
            cpu_cap = rng.uniform(4, 64)
            mem_cap = rng.uniform(8, 256)
            state[base + 0] = cpu_cap                        # cpu_capacity
            state[base + 1] = rng.uniform(0, cpu_cap)        # cpu_available
            state[base + 2] = mem_cap                        # memory_capacity
            state[base + 3] = rng.uniform(0, mem_cap)        # memory_available
            state[base + 4] = rng.choice([0, 1, 2, 4, 8])   # gpu_count
            state[base + 5] = rng.integers(0, 110)           # pod_count
            state[base + 6] = 110                            # pod_capacity
            state[base + 7] = rng.uniform(0.1, 5.0)         # cost_per_hour
            state[base + 8] = rng.integers(0, 3)             # zone index

        return state

    def _get_node_features(self, node_idx: int) -> np.ndarray:
        """Extract the feature vector for a single node."""
        base = node_idx * self.FEATURES_PER_NODE
        return self._state[base : base + self.FEATURES_PER_NODE]

    def _compute_reward(self, node_features: np.ndarray) -> float:
        """Compute reward for placing on the given node.

        TODO: mirror the Go reward.Reward() function logic.
        """
        cpu_util = 1.0 - (node_features[1] / max(node_features[0], 1e-6))
        mem_util = 1.0 - (node_features[3] / max(node_features[2], 1e-6))

        # Reward balanced utilization around 0.6.
        cpu_score = -((cpu_util - 0.6) ** 2)
        mem_score = -((mem_util - 0.6) ** 2)
        cost_penalty = -node_features[7] * 0.5

        return float((cpu_score + mem_score) * 5.0 + cost_penalty)
