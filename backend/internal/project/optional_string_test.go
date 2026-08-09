package project

import (
	"encoding/json"
	"testing"
)

func TestOptionalStringResolvePreservesOmittedValue(t *testing.T) {
	existing := "frontend"
	var payload struct {
		Value optionalString `json:"value"`
	}
	if err := json.Unmarshal([]byte(`{}`), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Value.Set {
		t.Fatal("omitted field must not be marked as set")
	}
	resolved := payload.Value.Resolve(&existing)
	if resolved == nil || *resolved != existing {
		t.Fatalf("Resolve() = %#v, want existing value %q", resolved, existing)
	}
}

func TestOptionalStringResolveClearsExplicitNull(t *testing.T) {
	existing := "frontend"
	var payload struct {
		Value optionalString `json:"value"`
	}
	if err := json.Unmarshal([]byte(`{"value":null}`), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Value.Set {
		t.Fatal("explicit null must be marked as set")
	}
	if resolved := payload.Value.Resolve(&existing); resolved != nil {
		t.Fatalf("Resolve() = %#v, want nil", resolved)
	}
}

func TestOptionalStringResolveReplacesExplicitString(t *testing.T) {
	existing := "frontend"
	var payload struct {
		Value optionalString `json:"value"`
	}
	if err := json.Unmarshal([]byte(`{"value":"apps/web"}`), &payload); err != nil {
		t.Fatal(err)
	}
	resolved := payload.Value.Resolve(&existing)
	if resolved == nil || *resolved != "apps/web" {
		t.Fatalf("Resolve() = %#v, want apps/web", resolved)
	}
}
