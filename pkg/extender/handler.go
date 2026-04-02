package extender

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	pb "github.com/mnorrsken/augur/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	extenderv1 "k8s.io/kube-scheduler/extender/v1"
)

// Handler implements the Kubernetes scheduler extender HTTP endpoints.
type Handler struct {
	agentConn   *grpc.ClientConn
	agentClient pb.AugurAgentClient
}

// NewHandler creates a Handler and dials the Python RL agent over gRPC.
func NewHandler() *Handler {
	agentAddr := os.Getenv("AUGUR_AGENT_ADDR")
	if agentAddr == "" {
		agentAddr = "localhost:50051"
	}

	conn, err := grpc.NewClient(agentAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to RL agent at %s: %v", agentAddr, err)
	}

	return &Handler{
		agentConn:   conn,
		agentClient: pb.NewAugurAgentClient(conn),
	}
}

// FilterHandler implements the /filter endpoint.
// It removes nodes that fail hard constraints (OPA policy, resource fit).
func (h *Handler) FilterHandler(r *http.Request) (any, error) {
	var args extenderv1.ExtenderArgs
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		return nil, fmt.Errorf("decode filter request: %w", err)
	}

	// TODO: evaluate OPA policy for each node.
	// TODO: remove nodes that violate hard constraints.

	result := &extenderv1.ExtenderFilterResult{
		Nodes: args.Nodes,
	}
	return result, nil
}

// PrioritizeHandler implements the /prioritize endpoint.
// It calls the RL agent to score each candidate node.
func (h *Handler) PrioritizeHandler(r *http.Request) (any, error) {
	var args extenderv1.ExtenderArgs
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		return nil, fmt.Errorf("decode prioritize request: %w", err)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Build gRPC request with node features.
	// TODO: extract real node features from args.Nodes.
	scoreReq := &pb.ScoreRequest{
		PodName:   args.Pod.Name,
		Namespace: args.Pod.Namespace,
	}

	scoreResp, err := h.agentClient.Score(ctx, scoreReq)
	if err != nil {
		log.Printf("agent scoring failed, using uniform scores: %v", err)
		return uniformScores(args), nil
	}

	priorities := make([]extenderv1.HostPriority, 0, len(scoreResp.NodeScores))
	for _, ns := range scoreResp.NodeScores {
		priorities = append(priorities, extenderv1.HostPriority{
			Host:  ns.NodeName,
			Score: int64(ns.Score),
		})
	}

	return priorities, nil
}

// uniformScores returns equal scores for all candidate nodes (fallback).
func uniformScores(args extenderv1.ExtenderArgs) []extenderv1.HostPriority {
	if args.Nodes == nil {
		return nil
	}
	priorities := make([]extenderv1.HostPriority, 0, len(args.Nodes.Items))
	for _, node := range args.Nodes.Items {
		priorities = append(priorities, extenderv1.HostPriority{
			Host:  node.Name,
			Score: 5,
		})
	}
	return priorities
}
