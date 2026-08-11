package caddy

import (
	"strings"
	"testing"
)

func TestRuntimeTargetFromInspectResolvesProjectRuntime(t *testing.T) {
	alias := runtimeRouteAlias(3456)
	raw := []byte(`[
  {
    "Id": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3456"}]},
      "Networks": {
        "mypaas-projects": {"Aliases": ["runtime"]}
      }
    }
  }
]`)

	got, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", alias, 3456)
	if err != nil {
		t.Fatalf("runtimeTargetFromInspect returned error: %v", err)
	}
	if got.ContainerID != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("container ID = %q", got.ContainerID)
	}
	if got.ContainerPort != "8080" {
		t.Fatalf("container port = %q, want 8080", got.ContainerPort)
	}
	if got.RoutingAttached {
		t.Fatal("new runtime should not already be attached to routing network")
	}
	if got.RoutingAliasPresent {
		t.Fatal("new runtime should not already have the routing alias")
	}
	if alias != "mypaas-port-3456" {
		t.Fatalf("alias = %q", alias)
	}
}

func TestRuntimeTargetFromInspectDetectsExistingRoutingAlias(t *testing.T) {
	alias := runtimeRouteAlias(3456)
	raw := []byte(`[
  {
    "Id": "aaaaaaaaaaaa0000000000000000000000000000000000000000000000000000",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3456"}]},
      "Networks": {
        "mypaas-projects": {"Aliases": ["runtime"]},
        "mypaas-routing": {"Aliases": ["runtime", "mypaas-port-3456"]}
      }
    }
  }
]`)

	got, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", alias, 3456)
	if err != nil {
		t.Fatalf("runtimeTargetFromInspect returned error: %v", err)
	}
	if !got.RoutingAttached || !got.RoutingAliasPresent {
		t.Fatalf("expected existing routing alias, got %+v", got)
	}
}

func TestRuntimeTargetFromInspectDetectsRoutingAttachmentWithoutAlias(t *testing.T) {
	alias := runtimeRouteAlias(3456)
	raw := []byte(`[
  {
    "Id": "bbbbbbbbbbbb1111111111111111111111111111111111111111111111111111",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3456"}]},
      "Networks": {
        "mypaas-projects": {"Aliases": ["runtime"]},
        "mypaas-routing": {"Aliases": ["runtime"]}
      }
    }
  }
]`)

	got, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", alias, 3456)
	if err != nil {
		t.Fatalf("runtimeTargetFromInspect returned error: %v", err)
	}
	if !got.RoutingAttached || got.RoutingAliasPresent {
		t.Fatalf("expected routing attachment without managed alias, got %+v", got)
	}
}

func TestRuntimeTargetFromInspectSelectsReplacementByAllocatedHostPort(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "aaaaaaaaaaaa0000000000000000000000000000000000000000000000000000",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3455"}]},
      "Networks": {"mypaas-projects": {"Aliases": ["old"]}}
    }
  },
  {
    "Id": "bbbbbbbbbbbb1111111111111111111111111111111111111111111111111111",
    "NetworkSettings": {
      "Ports": {"3000/tcp": [{"HostPort": "3456"}]},
      "Networks": {"mypaas-projects": {"Aliases": ["replacement"]}}
    }
  }
]`)

	got, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", runtimeRouteAlias(3456), 3456)
	if err != nil {
		t.Fatalf("runtimeTargetFromInspect returned error: %v", err)
	}
	if !strings.HasPrefix(got.ContainerID, "bbbbbbbbbbbb") || got.ContainerPort != "3000" {
		t.Fatalf("unexpected replacement target: %+v", got)
	}
}

func TestRuntimeTargetFromInspectSkipsSamePortOutsideProjectNetwork(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "cccccccccccc2222222222222222222222222222222222222222222222222222",
    "NetworkSettings": {
      "Ports": {"9000/tcp": [{"HostPort": "3456"}]},
      "Networks": {"other-network": {"Aliases": ["other"]}}
    }
  },
  {
    "Id": "dddddddddddd3333333333333333333333333333333333333333333333333333",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3456"}]},
      "Networks": {"mypaas-projects": {"Aliases": ["runtime"]}}
    }
  }
]`)

	got, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", runtimeRouteAlias(3456), 3456)
	if err != nil {
		t.Fatalf("runtimeTargetFromInspect returned error: %v", err)
	}
	if !strings.HasPrefix(got.ContainerID, "dddddddddddd") {
		t.Fatalf("unexpected project runtime: %+v", got)
	}
}

func TestRuntimeTargetFromInspectRequiresProjectNetworkAttachment(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "eeeeeeeeeeee4444444444444444444444444444444444444444444444444444",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3456"}]},
      "Networks": {"some-other-network": {"Aliases": ["runtime"]}}
    }
  }
]`)

	_, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", runtimeRouteAlias(3456), 3456)
	if err == nil {
		t.Fatal("expected missing project network to fail")
	}
	if !strings.Contains(err.Error(), "not attached to project network") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeTargetFromInspectRejectsUnknownPublishedPort(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "ffffffffffff5555555555555555555555555555555555555555555555555555",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3455"}]},
      "Networks": {"mypaas-projects": {"Aliases": ["runtime"]}}
    }
  }
]`)

	_, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", runtimeRouteAlias(3456), 3456)
	if err == nil {
		t.Fatal("expected unknown host port to fail")
	}
	if !strings.Contains(err.Error(), "no running container owns") {
		t.Fatalf("unexpected error: %v", err)
	}
}
