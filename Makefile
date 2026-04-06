.PHONY: build proto proto-go proto-py docker-build docker-build-agent deploy train agent-serve test lint fmt vet run clean

AGENT_IMAGE := augur-agent:latest
# Pass --gpus all when the Docker NVIDIA runtime is available, otherwise CPU-only.
GPU_FLAG := $(shell docker run --rm --gpus all hello-world >/dev/null 2>&1 && echo "--gpus all" || echo "")

# Go build
build:
	go build -o bin/augur-extender ./cmd/augur-extender

# Format Go code
fmt:
	go fmt ./...

# Vet Go code
vet:
	go vet ./...

# Run the extender locally (requires AUGUR_AGENT_ADDR to be set)
run: build
	./bin/augur-extender

# Generate protobuf stubs (Go + Python).
proto: proto-go proto-py

proto-go:
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/augur.proto

# Python proto stubs are generated inside the agent image at build time.
# This target lets you regenerate them locally into agent/ if needed.
proto-py: docker-build-agent
	docker run --rm \
		-v $(CURDIR)/agent:/out \
		$(AGENT_IMAGE) \
		sh -c "cp /app/augur_pb2.py /app/augur_pb2_grpc.py /out/"

# Build Docker images
docker-build: docker-build-agent
	docker build -t augur-extender:latest -f Dockerfile .

docker-build-agent:
	docker build -t $(AGENT_IMAGE) -f Dockerfile.agent .

# Deploy to Kubernetes via kustomize
deploy:
	kubectl create namespace augur --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -k deploy/

# Train the RL agent inside Docker with GPU passthrough.
# Model artifacts are written to ./agent/models/ on the host.
train: docker-build-agent
	docker run --rm $(GPU_FLAG) \
		-v $(CURDIR)/agent/models:/app/models \
		-v $(CURDIR)/agent/checkpoints:/app/checkpoints \
		-v $(CURDIR)/agent/tb_logs:/app/tb_logs \
		$(AGENT_IMAGE) \
		train.py --timesteps 100000

# Serve the RL agent via gRPC inside Docker.
agent-serve: docker-build-agent
	docker run --rm $(GPU_FLAG) \
		-p 50051:50051 \
		-v $(CURDIR)/agent/models:/app/models \
		$(AGENT_IMAGE) \
		serve.py --port 50051

# Run Go tests
test:
	go test ./...

# Lint Go code
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf bin/ agent/checkpoints/ agent/tb_logs/ agent/models/
