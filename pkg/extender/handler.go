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

	"github.com/mnorrsken/augur/pkg/spec"
	"github.com/mnorrsken/augur/pkg/state"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	extenderv1 "k8s.io/kube-scheduler/extender/v1"
)

// Handler implements the Kubernetes scheduler extender HTTP endpoints.
type Handler struct {
	agentConn   *grpc.ClientConn
	agentClient pb.AugurAgentClient
	opa         *spec.OPAClient
}

// NewHandler creates a Handler, dials the Python RL agent, and loads OPA policy.
func NewHandler() *Handler {
	agentAddr := os.Getenv("AUGUR_AGENT_ADDR")
	if agentAddr == "" {
		agentAddr = "localhost:50051"
	}

	policyPath := os.Getenv("AUGUR_POLICY_PATH")
	if policyPath == "" {
		policyPath = "config/policy.rego"
	}

	conn, err := grpc.NewClient(agentAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to RL agent at %s: %v", agentAddr, err)
	}

	return &Handler{
		agentConn:   conn,
		agentClient: pb.NewAugurAgentClient(conn),
		opa:         spec.NewOPAClient(policyPath),
	}
}

// FilterHandler implements the /filter endpoint.
// It removes nodes that fail hard constraints (OPA policy, resource fit).
func (h *Handler) FilterHandler(r *http.Request) (any, error) {
	var args extenderv1.ExtenderArgs
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		return nil, fmt.Errorf("decode filter request: %w", err)
	}

	if args.Nodes == nil {
		return &extenderv1.ExtenderFilterResult{}, nil
	}

	ws := spec.ParseFromAnnotations(args.Pod.Annotations)
	ws.Name = args.Pod.Name

	// First check workload-level constraints.
	ctx := r.Context()
	result, err := h.opa.Eval(ctx, ws)
	if err != nil {
		log.Printf("OPA workload eval error: %v", err)
		// Fail open — return all nodes.
		return &extenderv1.ExtenderFilterResult{Nodes: args.Nodes}, nil
	}
	if !result.Allowed {
		return &extenderv1.ExtenderFilterResult{
			Error: fmt.Sprintf("workload rejected: %v", result.Reasons),
		}, nil
	}

	// Per-node filtering: check each candidate node against OPA constraints.
	nodeFeatures := state.FromNodeList(args.Nodes.Items)
	var filteredItems []interface{ GetName() string }
	kept := make([]int, 0, len(args.Nodes.Items))
	var failedNodes map[string]string

	for i := range nodeFeatures {
		nf := &nodeFeatures[i]
		evalResult, evalErr := h.opa.EvalNode(ctx, ws, nf)
		if evalErr != nil {
			log.Printf("OPA node eval error for %s: %v", nf.NodeName, evalErr)
			// Fail open — keep the node.
			kept = append(kept, i)
			continue
		}
		if evalResult.Allowed {
			kept = append(kept, i)
		} else {
			if failedNodes == nil {
				failedNodes = make(map[string]string)
			}
			failedNodes[nf.NodeName] = fmt.Sprintf("%v", evalResult.Reasons)
			_ = filteredItems // suppress unused warning
		}
	}

	// Build the filtered node list.
	filteredNodeList := args.Nodes.DeepCopy()
	filteredNodeList.Items = nil
	for _, idx := range kept {
		filteredNodeList.Items = append(filteredNodeList.Items, args.Nodes.Items[idx])
	}

	resp := &extenderv1.ExtenderFilterResult{
		Nodes:       filteredNodeList,
		FailedNodes: failedNodes,
	}
	return resp, nil
}

// PrioritizeHandler implements the /prioritize endpoint.
// It calls the RL agent to score each candidate node.
func (h *Handler) PrioritizeHandler(r *http.Request) (any, error) {
	var args extenderv1.ExtenderArgs
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		return nil, fmt.Errorf("decode prioritize request: %w", err)
	}

	if args.Nodes == nil || len(args.Nodes.Items) == 0 {
		return []extenderv1.HostPriority{}, nil
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	ws := spec.ParseFromAnnotations(args.Pod.Annotations)
	nodeFeatures := state.FromNodeList(args.Nodes.Items)

	// Build gRPC request with real node features.
	scoreReq := &pb.ScoreRequest{
		PodName:   args.Pod.Name,
		Namespace: args.Pod.Namespace,
		Intent:    ws.Intent,
	}

	for i := range nodeFeatures {
		nf := &nodeFeatures[i]
		scoreReq.Nodes = append(scoreReq.Nodes, &pb.NodeFeatures{
			NodeName:       nf.NodeName,
			CpuCapacity:    nf.CPUCapacity,
			CpuAvailable:   nf.CPUAvailable,
			MemoryCapacity: nf.MemoryCapacity,
			MemoryAvailable: nf.MemoryAvailable,
			GpuCount:       nf.GPUCount,
			PodCount:       nf.PodCount,
			PodCapacity:    nf.PodCapacity,
			CostPerHour:    nf.CostPerHour,
			Zone:           nf.Zone,
		})
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
