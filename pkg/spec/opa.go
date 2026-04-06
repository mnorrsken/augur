package spec

import (
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/mnorrsken/augur/pkg/state"
)

// OPAClient evaluates workload specs against OPA policies.
type OPAClient struct {
	policyPath string
	query      string
}

// NewOPAClient creates a new OPA client with the given policy path.
func NewOPAClient(policyPath string) *OPAClient {
	return &OPAClient{
		policyPath: policyPath,
		query:      "data.augur.deny",
	}
}

// EvalResult holds the output of a policy evaluation.
type EvalResult struct {
	Allowed bool     `json:"allowed"`
	Reasons []string `json:"reasons,omitempty"`
}

// NodeInput is the OPA input document when evaluating per-node constraints.
type NodeInput struct {
	Spec *WorkloadSpec       `json:"spec"`
	Node *state.NodeFeatures `json:"node"`
}

// Eval evaluates the given workload spec against the loaded OPA policy.
// It returns whether the workload is allowed and any denial reasons.
func (c *OPAClient) Eval(ctx context.Context, spec *WorkloadSpec) (*EvalResult, error) {
	input := map[string]any{"spec": spec}
	return c.eval(ctx, input)
}

// EvalNode evaluates whether a specific node is acceptable for the workload
// by providing both the spec and node features to OPA.
func (c *OPAClient) EvalNode(ctx context.Context, spec *WorkloadSpec, node *state.NodeFeatures) (*EvalResult, error) {
	input := NodeInput{Spec: spec, Node: node}
	return c.eval(ctx, input)
}

func (c *OPAClient) eval(ctx context.Context, input any) (*EvalResult, error) {
	r := rego.New(
		rego.Query(c.query),
		rego.Load([]string{c.policyPath}, nil),
		rego.Input(input),
	)

	rs, err := r.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("opa eval: %w", err)
	}

	result := &EvalResult{Allowed: true}

	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return result, nil
	}

	// The deny rule produces a set of strings. OPA returns it as []interface{}.
	val := rs[0].Expressions[0].Value
	switch reasons := val.(type) {
	case []any:
		for _, reason := range reasons {
			if s, ok := reason.(string); ok {
				result.Reasons = append(result.Reasons, s)
			}
		}
	}

	if len(result.Reasons) > 0 {
		result.Allowed = false
	}
	return result, nil
}
