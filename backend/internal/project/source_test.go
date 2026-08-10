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
	valid := []string{
		"nginx:latest",
		"user/my-app:1.2.0",
		"ghcr.io/user/my-app:latest",
		"registry.example.com/team/app@sha256:0123456789abcdef",
	}
	for _, value := range valid {
		t.Run("valid_"+value, func(t *testing.T) {
			got, err := normalizeImageRef(&value)
			if err != nil {
				t.Fatalf("normalizeImageRef() error = %v", err)
			}
			if got == nil || *got != value {
				t.Fatalf("normalizeImageRef() = %#v, want %q", got, value)
			}
		})
	}

	invalid := []string{"", "   ", "-it", "https://ghcr.io/user/app:latest", "ghcr.io/user/app latest"}
	for _, value := range invalid {
		t.Run("invalid_"+value, func(t *testing.T) {
			if _, err := normalizeImageRef(&value); err == nil {
				t.Fatal("normalizeImageRef() expected validation error")
			}
		})
	}
}
