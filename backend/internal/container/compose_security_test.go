package container

import (
	"strings"
	"testing"
)

func TestSanitizeComposeConfigRejectsHostEscapeFeatures(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "privileged", raw: `{"services":{"app":{"image":"alpine","privileged":true}}}`, want: "privileged"},
		{name: "api socket", raw: `{"services":{"app":{"image":"alpine","use_api_socket":true}}}`, want: "use_api_socket"},
		{name: "provider binary", raw: `{"services":{"db":{"provider":{"type":"evil-provider"}}}}`, want: "provider services"},
		{name: "host network", raw: `{"services":{"app":{"image":"alpine","network_mode":"host"}}}`, want: "network_mode"},
		{name: "container pid namespace", raw: `{"services":{"app":{"image":"alpine","pid":"container:host-target"}}}`, want: "pid"},
		{name: "docker socket bind", raw: `{"services":{"app":{"image":"alpine","volumes":[{"type":"bind","source":"/var/run/docker.sock","target":"/var/run/docker.sock"}]}}}`, want: "bind mounts"},
		{name: "host root bind", raw: `{"services":{"app":{"image":"alpine","volumes":[{"type":"bind","source":"/","target":"/host"}]}}}`, want: "bind mounts"},
		{name: "device", raw: `{"services":{"app":{"image":"alpine","devices":["/dev/kvm:/dev/kvm"]}}}`, want: "devices"},
		{name: "gpu", raw: `{"services":{"app":{"image":"alpine","gpus":"all"}}}`, want: "gpus"},
		{name: "added capabilities", raw: `{"services":{"app":{"image":"alpine","cap_add":["SYS_ADMIN"]}}}`, want: "cap_add"},
		{name: "custom runtime", raw: `{"services":{"app":{"image":"alpine","runtime":"kata-custom"}}}`, want: "custom runtime"},
		{name: "external links", raw: `{"services":{"app":{"image":"alpine","external_links":["other-project-db"]}}}`, want: "external_links"},
		{name: "privileged lifecycle hook", raw: `{"services":{"app":{"image":"alpine","post_start":[{"command":"id","privileged":true}]}}}`, want: "privileged lifecycle"},
		{name: "privileged build", raw: `{"services":{"app":{"build":{"context":".","privileged":true}}}}`, want: "privileged image builds"},
		{name: "host build network", raw: `{"services":{"app":{"build":{"context":".","network":"host"}}}}`, want: "build network=host"},
		{name: "build entitlement", raw: `{"services":{"app":{"build":{"context":".","entitlements":["security.insecure"]}}}}`, want: "build entitlements"},
		{name: "additional build context", raw: `{"services":{"app":{"build":{"context":".","additional_contexts":{"host":"/mypaas"}}}}}`, want: "build additional_contexts"},
		{name: "build ssh", raw: `{"services":{"app":{"build":{"context":".","ssh":["default"]}}}}`, want: "build ssh"},
		{name: "external volume", raw: `{"services":{"app":{"image":"alpine","volumes":[{"type":"volume","source":"shared","target":"/data"}]}},"volumes":{"shared":{"external":true}}}`, want: "must not be external"},
		{name: "external network", raw: `{"services":{"app":{"image":"alpine"}},"networks":{"host-shared":{"external":true}}}`, want: "must not be external"},
		{name: "file backed secrets", raw: `{"services":{"app":{"image":"alpine"}},"secrets":{"host":{"file":"/etc/shadow"}}}`, want: "configs/secrets"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sanitizeComposeConfig([]byte(tt.raw))
			if err == nil {
				t.Fatal("sanitizeComposeConfig() unexpectedly accepted unsafe config")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("sanitizeComposeConfig() error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestSanitizeComposeConfigAllowsNamedVolumesAndSafeOptions(t *testing.T) {
	raw := []byte(`{
		"services": {
			"app": {
				"image": "alpine",
				"runtime": "runc",
				"security_opt": ["no-new-privileges:true"],
				"volumes": [{"type":"volume","source":"data","target":"/data"}],
				"ports": [{"target":8080,"published":"8080"}]
			}
		},
		"volumes": {"data": {"driver":"local"}},
		"networks": {"default": {"driver":"bridge"}}
	}`)

	out, err := sanitizeComposeConfig(raw)
	if err != nil {
		t.Fatalf("sanitizeComposeConfig() safe config error = %v", err)
	}
	if strings.Contains(string(out), `"ports"`) {
		t.Fatalf("sanitized output still contains host ports: %s", out)
	}
}
