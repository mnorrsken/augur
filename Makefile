.PHONY: build proto docker-build deploy train test lint clean

# Go build
build:
	go build -o bin/augur-extender ./cmd/augur-extender

# Generate protobuf Go and Python stubs
proto:
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/augur.proto
	python -m grpc_tools.protoc \
		-Iproto \
		--python_out=agent \
		--grpc_python_out=agent \
		proto/augur.proto

# Build Docker images
docker-build:
	docker build -t augur-extender:latest -f Dockerfile .
	docker build -t augur-agent:latest -f Dockerfile.agent .

# Deploy to Kubernetes via kustomize
deploy:
	kubectl create namespace augur --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -k deploy/

# Train the RL agent locally
train:
	cd agent && python train.py --timesteps 100000

# Run Go tests
test:
	go test ./...

# Lint Go code
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf bin/ agent/checkpoints/ agent/tb_logs/ agent/models/
