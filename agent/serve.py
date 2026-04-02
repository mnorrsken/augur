"""gRPC server exposing the trained RL agent for online scoring.

The Go scheduler extender calls Score(ScoreRequest) and receives
per-node scores from the loaded PPO model.
"""

import argparse
import os
from concurrent import futures

import grpc
import numpy as np
from stable_baselines3 import PPO

# Generated proto stubs — run `make proto` to generate.
import augur_pb2
import augur_pb2_grpc


class AugurAgentServicer(augur_pb2_grpc.AugurAgentServicer):
    """Implements the AugurAgent gRPC service."""

    def __init__(self, model_path: str) -> None:
        if os.path.exists(model_path + ".zip") or os.path.exists(model_path):
            self.model = PPO.load(model_path)
            print(f"Loaded model from {model_path}")
        else:
            self.model = None
            print(f"WARNING: no model at {model_path}, returning uniform scores")

    def Score(self, request: augur_pb2.ScoreRequest, context) -> augur_pb2.ScoreResponse:
        """Score each candidate node for the given pod."""
        response = augur_pb2.ScoreResponse()

        for node in request.nodes:
            obs = np.array([
                node.cpu_capacity,
                node.cpu_available,
                node.memory_capacity,
                node.memory_available,
                node.gpu_count,
                node.pod_count,
                node.pod_capacity,
                node.cost_per_hour,
                node.zone,
            ], dtype=np.float32)

            if self.model is not None:
                # TODO: the model expects the full observation space from the env;
                # adapt this to match the actual observation shape.
                action, _ = self.model.predict(obs, deterministic=True)
                score = float(action) if np.isscalar(action) else float(action[0])
            else:
                score = 5.0  # uniform fallback

            ns = augur_pb2.NodeScore(
                node_name=node.node_name,
                score=score,
            )
            response.node_scores.append(ns)

        return response


def serve(port: int = 50051, model_path: str = "models/augur_ppo") -> None:
    """Start the gRPC server."""
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    augur_pb2_grpc.add_AugurAgentServicer_to_server(
        AugurAgentServicer(model_path), server
    )
    server.add_insecure_port(f"[::]:{port}")
    server.start()
    print(f"Augur agent gRPC server listening on :{port}")
    server.wait_for_termination()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Serve Augur RL agent via gRPC")
    parser.add_argument("--port", type=int, default=50051)
    parser.add_argument("--model-path", type=str, default="models/augur_ppo")
    args = parser.parse_args()

    serve(port=args.port, model_path=args.model_path)
