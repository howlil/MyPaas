package container

import "testing"

func TestRepoDigestFromInspect(t *testing.T) {
	raw := []byte(`[{"RepoDigests":["ghcr.io/example/app@sha256:abcdef"],"Id":"sha256:123"}]`)
	got, err := repoDigestFromInspect(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != "ghcr.io/example/app@sha256:abcdef" {
		t.Fatalf("repoDigestFromInspect() = %q", got)
	}
}

func TestRepoDigestFromInspectAllowsMissingDigest(t *testing.T) {
	got, err := repoDigestFromInspect([]byte(`[{"RepoDigests":[]}]`))
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("repoDigestFromInspect() = %q, want empty", got)
	}
}
