# Augur

AI-powered Kubernetes scheduler extender with reinforcement learning-based node scoring and intent-based workload placement.

## Architecture

```
┌────────────────┐     HTTP /filter      ┌───────────────────┐
│  kube-scheduler │────/prioritize───────▶│  augur-extender   │
│                │◀──────────────────────│  (Go, port 8888)  │
└────────────────┘                       └──────┬────────────┘
                                                │ gRPC Score()
                                                ▼
                                         ┌───────────────────┐
                                         │  augur-agent       │
                                         │  (Python, port     │
                                         │   50051, PPO model)│
                                         └───────────────────┘
```

**augur-extender** is a Kubernetes scheduler extender that intercepts scheduling
decisions via `/filter` and `/prioritize` webhooks. It evaluates hard constraints
using OPA policies and delegates soft scoring to the **augur-agent**, a Python
gRPC service running a PPO-trained reinforcement learning model.

### Key Components

| Component | Language | Path | Purpose |
|-----------|----------|------|---------|
| Extender HTTP server | Go | `cmd/augur-extender/` | Scheduler webhook endpoints |
| WorkloadSpec + OPA | Go | `pkg/spec/` | Intent parsing and policy evaluation |
| Reward function | Go | `pkg/reward/` | Scalar reward from placement outcomes |
| State encoder | Go | `pkg/state/` | Node feature extraction for RL agent |
| Extender handlers | Go | `pkg/extender/` | Filter/Prioritize logic + gRPC client |
| RL training | Python | `agent/train.py` | PPO training loop (stable-baselines3) |
| RL serving | Python | `agent/serve.py` | gRPC server for online scoring |
| Gym environment | Python | `agent/env.py` | Simulated cluster for training |
| Offline replay | Go | `sim/replay.go` | Replay Prometheus snapshots |

## Prerequisites

- Go 1.22+
- Python 3.11+
- protoc with `protoc-gen-go` and `protoc-gen-go-grpc` plugins
- Docker (for container builds)
- Access to a Kubernetes cluster (k3s for local dev)

## Quickstart

```bash
# Build the Go extender
make build

# Generate protobuf stubs (requires protoc)
make proto

# Train the RL agent locally
make train

# Build Docker images
make docker-build

# Deploy to Kubernetes
make deploy
```

## Local Development with k3s

```bash
# Install k3s
curl -sfL https://get.k3s.io | sh -

# Copy the scheduler config
sudo cp config/scheduler-config.yaml /var/lib/rancher/k3s/server/

# Restart k3s with the custom scheduler config
sudo systemctl restart k3s

# Deploy Augur
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
make deploy
```

## Proto Generation

If `protoc` is not available, install it:

```bash
# Install protoc
apt-get install -y protobuf-compiler
# or: brew install protobuf

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Install Python plugin
pip install grpcio-tools

# Generate stubs
make proto
```

## Project Status

This is the initial skeleton. See [CLAUDE.md](CLAUDE.md) for the phased build plan.

## License

See [LICENSE](LICENSE).
