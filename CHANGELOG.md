# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- **Phase 2: State encoder** — `pkg/state/encoder.go` extracts a 9-feature vector
  (CPU capacity/available, memory capacity/available in GiB, GPU count, pod count,
  pod capacity, cost/hr, zone index) from real `v1.Node` objects via `FromNodeList()`.
- **Phase 2: Reward function** — `pkg/reward/reward.go` implements a composite reward
  with eviction hard-cutoff (−10), latency penalty, utilisation balance (peak at 60%),
  cost penalty, and intent-match bonuses (`gpu-intensive`, `low-latency`, `cost-sensitive`).
- **Phase 2: WorkloadSpec** — `pkg/spec/spec.go` parses `WorkloadSpec` from pod
  annotations (`augur.io/intent`, `augur.io/max-cost`, `augur.io/zones`,
  `augur.io/priority`, `augur.io/replicas`, `augur.io/gpu-request`).
- **Phase 2: OPA evaluation** — `pkg/spec/opa.go` wires full OPA policy evaluation;
  supports both workload-level and per-node constraint checks.
- **Phase 2: OPA policy rules** — `config/policy.rego` enforces zone affinity, cost
  ceiling, and GPU availability constraints.
- **Extender integration** — `pkg/extender/handler.go` wires OPA into `/filter` and
  `/prioritize` handlers; loads policy path from `AUGUR_POLICY_PATH` env var
  (default: `config/policy.rego`).
- **Unit tests** — `pkg/reward`, `pkg/spec`, `pkg/state` packages each have passing
  test suites (`*_test.go`).
- **Makefile targets** — added `fmt` (`go fmt ./...`), `vet` (`go vet ./...`), and
  `run` (builds then runs the extender binary) per Go build conventions.
- **CLAUDE.md** — documented wiki skill lookup convention for implementation tasks.

## [0.2.0] — 2026-03-01

### Added

- Docker-based training and agent-serve targets in `Makefile` with GPU passthrough
  (`--gpus all` when NVIDIA runtime is available, CPU fallback otherwise).
- `Dockerfile.agent` and `docker-build-agent` Make target.

## [0.1.0] — 2026-02-01

### Added

- Initial Augur skeleton: Go HTTP extender with stub `/filter` and `/prioritize`
  endpoints, gRPC contract (`proto/augur.proto`), and `KubeSchedulerConfiguration`.
- Python RL agent scaffolding (`agent/env.py`, `agent/train.py`, `agent/serve.py`).
- Offline simulation harness (`sim/replay.go`).
- `config/scheduler-config.yaml` for k3s extender registration.
