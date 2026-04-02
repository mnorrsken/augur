package augur

# deny is a set of reasons the workload should be rejected.

deny["replicas exceeds maximum of 50"] {
    input.spec.replicas > 50
}

deny["priority must be non-negative"] {
    input.spec.priority < 0
}

# TODO: add zone-affinity constraints.
# TODO: add cost-ceiling enforcement.
# TODO: add GPU-request vs node-GPU validation.
