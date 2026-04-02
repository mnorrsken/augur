"""PPO training loop for the Augur RL scheduling agent.

Uses stable-baselines3 to train a PPO policy on the custom
KubeSchedulerEnv gym environment.
"""

import argparse
import os

from stable_baselines3 import PPO
from stable_baselines3.common.callbacks import CheckpointCallback

from env import KubeSchedulerEnv


def make_env(num_nodes: int = 20) -> KubeSchedulerEnv:
    """Create and return the cluster simulation environment."""
    return KubeSchedulerEnv(num_nodes=num_nodes)


def train(
    total_timesteps: int = 100_000,
    num_nodes: int = 20,
    save_path: str = "models/augur_ppo",
) -> None:
    """Train the PPO agent."""
    env = make_env(num_nodes=num_nodes)

    model = PPO(
        policy="MlpPolicy",
        env=env,
        verbose=1,
        learning_rate=3e-4,
        n_steps=2048,
        batch_size=64,
        n_epochs=10,
        gamma=0.99,
        gae_lambda=0.95,
        clip_range=0.2,
        tensorboard_log="./tb_logs/",
    )

    checkpoint_cb = CheckpointCallback(
        save_freq=10_000,
        save_path="./checkpoints/",
        name_prefix="augur",
    )

    # TODO: add custom callback for Prometheus metrics export.

    model.learn(total_timesteps=total_timesteps, callback=checkpoint_cb)

    os.makedirs(os.path.dirname(save_path), exist_ok=True)
    model.save(save_path)
    print(f"Model saved to {save_path}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Train Augur RL agent")
    parser.add_argument("--timesteps", type=int, default=100_000)
    parser.add_argument("--nodes", type=int, default=20)
    parser.add_argument("--save-path", type=str, default="models/augur_ppo")
    args = parser.parse_args()

    train(
        total_timesteps=args.timesteps,
        num_nodes=args.nodes,
        save_path=args.save_path,
    )
