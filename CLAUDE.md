# Augur — Claude Code Project Notes

## Overview

Augur is an AI-powered Kubernetes scheduler extender. A Go HTTP server intercepts
kube-scheduler decisions and delegates scoring to a Python RL agent over gRPC.

## Build Phases

### Phase 1: Extender Skeleton
**Goal:** Get the Go HTTP server running and responding to kube-scheduler webhooks.
**Files:**
- `cmd/augur-extender/main.go` — HTTP server setup
- `pkg/extender/handler.go` — FilterHandler and PrioritizeHandler
- `proto/augur.proto` — gRPC contract definition
- `proto/augur.pb.go`, `proto/augur_grpc.pb.go` — generated stubs
- `config/scheduler-config.yaml` — scheduler extender registration

**Done when:** `make build` produces a binary, and the extender responds to
`/filter` and `/prioritize` with stub responses against a k3s cluster.

### Phase 2: State + Reward
**Goal:** Implement node feature extraction and the reward function.
**Files:**
- `pkg/state/encoder.go` — `FromNodeList()` extracting real features from `v1.Node` objects
- `pkg/reward/reward.go` — tune reward weights, add intent-match bonus
- `pkg/spec/spec.go` — `ParseFromAnnotations()` parsing pod annotations into WorkloadSpec
- `pkg/spec/opa.go` — full OPA evaluation wiring

**Done when:** given a real node list, the encoder produces a correct feature matrix;
the reward function returns meaningful values for test placement outcomes.

### Phase 3: RL Agent
**Goal:** Train a PPO agent and serve it for online scoring.
**Files:**
- `agent/env.py` — replace random state generation with Prometheus snapshot replay
- `agent/train.py` — tune hyperparameters, add Prometheus metrics callback
- `agent/serve.py` — load trained model, handle Score RPCs
- `sim/replay.go` — offline replay harness for evaluation

**Done when:** `make train` produces a model that scores better than uniform random
on historical scheduling traces; `agent/serve.py` responds to gRPC calls.

### Phase 4: Intent Layer
**Goal:** Parse free-form workload intents and feed them into scoring.
**Files:**
- `pkg/spec/spec.go` — extend WorkloadSpec with parsed intent features
- `pkg/spec/opa.go` — add intent-aware policy rules
- `config/policy.rego` — zone, cost, and GPU constraint rules
- `pkg/extender/handler.go` — pass intent features to the agent
- `agent/env.py` — include intent in observation space

**Done when:** a pod annotated with `augur.io/intent: gpu-intensive` gets
preferentially scheduled to GPU nodes by the RL agent.

## gRPC Contract

The Go extender calls the Python agent via gRPC using the `AugurAgent` service
defined in `proto/augur.proto`:

```
service AugurAgent {
  rpc Score(ScoreRequest) returns (ScoreResponse);
}
```

- **ScoreRequest** contains `pod_name`, `namespace`, `intent`, and a list of
  `NodeFeatures` messages (one per candidate node).
- **ScoreResponse** contains a list of `NodeScore` messages, each pairing a
  `node_name` with a `float score`.
- The extender maps these scores to `HostPriority` values returned to kube-scheduler.

The agent gRPC server listens on port **50051** by default (env: `AUGUR_AGENT_ADDR`).

## Running Locally Against k3s

```bash
# 1. Install k3s
curl -sfL https://get.k3s.io | sh -

# 2. Build and run the extender
make build
AUGUR_AGENT_ADDR=localhost:50051 ./bin/augur-extender

# 3. Run the Python agent (in another terminal)
cd agent
pip install -r requirements.txt
python serve.py --port 50051

# 4. Configure k3s to use the extender
sudo cp config/scheduler-config.yaml /var/lib/rancher/k3s/server/
sudo systemctl restart k3s

# 5. Schedule a test pod and observe scoring
kubectl run test-pod --image=nginx
kubectl logs -l app=augur-extender  # check extender logs
```

## Conventions

- Go code uses standard `go fmt` formatting.
- Proto stubs are generated into `proto/` (Go) and `agent/` (Python).
- All Kubernetes manifests live in `deploy/` and are managed by Kustomize.
- OPA policies live in `config/` and are loaded at runtime by the extender.
- The reward function in Go (`pkg/reward/reward.go`) and the env reward in Python
  (`agent/env.py`) must be kept in sync — changes to reward logic should update both.
- Environment variables: `AUGUR_LISTEN_ADDR` (extender port), `AUGUR_AGENT_ADDR` (agent endpoint).
