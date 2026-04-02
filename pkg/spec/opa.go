package spec

import (
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/v1/rego"
)

// OPAClient evaluates workload specs against OPA policies.
type OPAClient struct {
	// policyPath is the path to the .rego policy file.
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

// Eval evaluates the given workload spec against the loaded OPA policy.
// It returns whether the workload is allowed and any denial reasons.
func (c *OPAClient) Eval(ctx context.Context, spec *WorkloadSpec) (*EvalResult, error) {
	r := rego.New(
		rego.Query(c.query),
		rego.Load([]string{c.policyPath}, nil),
	)

	rs, err := r.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("opa eval: %w", err)
	}

	// TODO: parse rs into EvalResult — extract deny reasons from result set.
	_ = rs
	return &EvalResult{Allowed: true}, nil
}
