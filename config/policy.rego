package augur

# deny is a set of reasons the workload or node placement should be rejected.

# --- Workload-level constraints (evaluated without node context) ---

deny contains "replicas exceeds maximum of 50" if {
    input.spec.replicas > 50
}

deny contains "priority must be non-negative" if {
    input.spec.priority < 0
}

# --- Node-level constraints (evaluated per candidate node) ---

# Zone affinity: reject nodes outside the required zones.
deny contains msg if {
    count(input.spec.requiredZones) > 0
    input.node
    not zone_allowed
    msg := sprintf("node zone %q not in required zones %v", [input.node.ZoneName, input.spec.requiredZones])
}

zone_allowed if {
    some zone in input.spec.requiredZones
    input.node.ZoneName == zone
}

# Cost ceiling: reject nodes whose hourly cost exceeds the workload budget.
deny contains msg if {
    input.spec.maxCostPerHour > 0
    input.node
    input.node.CostPerHour > input.spec.maxCostPerHour
    msg := sprintf("node cost $%.2f/hr exceeds budget $%.2f/hr", [input.node.CostPerHour, input.spec.maxCostPerHour])
}

# GPU request: reject nodes that don't have enough GPUs.
deny contains msg if {
    input.spec.gpuRequest > 0
    input.node
    input.node.GPUCount < input.spec.gpuRequest
    msg := sprintf("node has %.0f GPUs but workload requires %d", [input.node.GPUCount, input.spec.gpuRequest])
}
