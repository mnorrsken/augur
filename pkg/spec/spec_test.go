package spec

import "testing"

func TestParseFromAnnotations_Intent(t *testing.T) {
	ws := ParseFromAnnotations(map[string]string{
		AnnotationIntent: "gpu-intensive",
	})
	if ws.Intent != "gpu-intensive" {
		t.Errorf("Intent = %q, want %q", ws.Intent, "gpu-intensive")
	}
}

func TestParseFromAnnotations_MaxCost(t *testing.T) {
	ws := ParseFromAnnotations(map[string]string{
		AnnotationMaxCost: "3.75",
	})
	if ws.MaxCostPerHour != 3.75 {
		t.Errorf("MaxCostPerHour = %v, want 3.75", ws.MaxCostPerHour)
	}
}

func TestParseFromAnnotations_InvalidMaxCost(t *testing.T) {
	ws := ParseFromAnnotations(map[string]string{
		AnnotationMaxCost: "expensive",
	})
	if ws.MaxCostPerHour != 0 {
		t.Errorf("MaxCostPerHour = %v, want 0 for invalid value", ws.MaxCostPerHour)
	}
}

func TestParseFromAnnotations_Zones(t *testing.T) {
	ws := ParseFromAnnotations(map[string]string{
		AnnotationZones: "us-east-1a, us-east-1b, us-east-1c",
	})
	want := []string{"us-east-1a", "us-east-1b", "us-east-1c"}
	if len(ws.RequiredZones) != len(want) {
		t.Fatalf("RequiredZones len = %d, want %d", len(ws.RequiredZones), len(want))
	}
	for i, z := range want {
		if ws.RequiredZones[i] != z {
			t.Errorf("RequiredZones[%d] = %q, want %q", i, ws.RequiredZones[i], z)
		}
	}
}

func TestParseFromAnnotations_ZonesSingleEntry(t *testing.T) {
	ws := ParseFromAnnotations(map[string]string{
		AnnotationZones: "eu-west-1a",
	})
	if len(ws.RequiredZones) != 1 || ws.RequiredZones[0] != "eu-west-1a" {
		t.Errorf("RequiredZones = %v, want [eu-west-1a]", ws.RequiredZones)
	}
}

func TestParseFromAnnotations_Priority(t *testing.T) {
	ws := ParseFromAnnotations(map[string]string{
		AnnotationPriority: "10",
	})
	if ws.Priority != 10 {
		t.Errorf("Priority = %d, want 10", ws.Priority)
	}
}

func TestParseFromAnnotations_NegativePriority(t *testing.T) {
	ws := ParseFromAnnotations(map[string]string{
		AnnotationPriority: "-5",
	})
	if ws.Priority != -5 {
		t.Errorf("Priority = %d, want -5", ws.Priority)
	}
}

func TestParseFromAnnotations_Replicas(t *testing.T) {
	ws := ParseFromAnnotations(map[string]string{
		AnnotationReplicas: "3",
	})
	if ws.Replicas != 3 {
		t.Errorf("Replicas = %d, want 3", ws.Replicas)
	}
}

func TestParseFromAnnotations_GPURequest(t *testing.T) {
	ws := ParseFromAnnotations(map[string]string{
		AnnotationGPURequest: "2",
	})
	if ws.GPURequest != 2 {
		t.Errorf("GPURequest = %d, want 2", ws.GPURequest)
	}
}

func TestParseFromAnnotations_Empty(t *testing.T) {
	ws := ParseFromAnnotations(map[string]string{})
	if ws.Intent != "" || ws.MaxCostPerHour != 0 || len(ws.RequiredZones) != 0 ||
		ws.Priority != 0 || ws.Replicas != 0 || ws.GPURequest != 0 {
		t.Errorf("empty annotations should produce zero-value WorkloadSpec, got %+v", ws)
	}
}

func TestParseFromAnnotations_AllFields(t *testing.T) {
	ws := ParseFromAnnotations(map[string]string{
		AnnotationIntent:     "low-latency",
		AnnotationMaxCost:    "1.5",
		AnnotationZones:      "us-west-2a,us-west-2b",
		AnnotationPriority:   "5",
		AnnotationReplicas:   "2",
		AnnotationGPURequest: "0",
	})
	if ws.Intent != "low-latency" {
		t.Errorf("Intent = %q", ws.Intent)
	}
	if ws.MaxCostPerHour != 1.5 {
		t.Errorf("MaxCostPerHour = %v", ws.MaxCostPerHour)
	}
	if len(ws.RequiredZones) != 2 {
		t.Errorf("RequiredZones = %v", ws.RequiredZones)
	}
	if ws.Priority != 5 {
		t.Errorf("Priority = %d", ws.Priority)
	}
	if ws.Replicas != 2 {
		t.Errorf("Replicas = %d", ws.Replicas)
	}
}
