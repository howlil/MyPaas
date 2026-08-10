package project

import "testing"

func TestNormalizeSourceType(t *testing.T) {
	tests := []struct {
		name       string
		sourceType string
		deployMode string
		want       string
		wantErr    bool
	}{
		{name: "legacy git default", want: SourceTypeGit},
		{name: "image infers registry", deployMode: "image", want: SourceTypeRegistry},
		{name: "explicit git", sourceType: " git ", want: SourceTypeGit},
		{name: "explicit registry", sourceType: "REGISTRY", want: SourceTypeRegistry},
		{name: "unknown", sourceType: "upload", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeSourceType(tt.sourceType, tt.deployMode)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeSourceType() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("normalizeSourceType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeImageRef(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "docker hub", value: "nginx:latest", want: "nginx:latest"},
		{name: "namespaced tag", value: "user/my-app:1.2.0", want: "user/my-app:1.2.0"},
		{name: "ghcr", value: "ghcr.io/user/my-app:latest", want: "ghcr.io/user/my-app:latest"},
		{name: "digest", value: "registry.example.com/team/app@sha256:0123456789abcdef", want: "registry.example.com/team/app@sha256:0123456789abcdef"},
		{name: "copied docker pull", value: "docker pull ghcr.io/user/my-app:latest", want: "ghcr.io/user/my-app:latest"},
		{name: "copied docker pull extra spacing", value: "  docker   pull   nginx:latest  ", want: "nginx:latest"},
		{name: "copied podman pull", value: "podman pull registry.example.com/team/app:prod", want: "registry.example.com/team/app:prod"},
		{name: "empty", value: "", wantErr: true},
		{name: "whitespace", value: "   ", wantErr: true},
		{name: "option", value: "-it", wantErr: true},
		{name: "url", value: "https://ghcr.io/user/app:latest", wantErr: true},
		{name: "raw whitespace", value: "ghcr.io/user/app latest", wantErr: true},
		{name: "extra command argument", value: "docker pull nginx:latest extra", wantErr: true},
		{name: "shell-like suffix", value: "docker pull nginx:latest && echo nope", wantErr: true},
		{name: "newline", value: "docker pull nginx:latest\necho nope", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeImageRef(&tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeImageRef() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got == nil || *got != tt.want {
				t.Fatalf("normalizeImageRef() = %#v, want %q", got, tt.want)
			}
		})
	}
}
