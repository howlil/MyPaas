package container

import (
	"fmt"
	"strings"
)

// validateComposeSecurity enforces the host-isolation boundary on the fully
// rendered Docker Compose configuration. Detection/UI warnings are not a
// security boundary; this validation runs on the exact config that is about to
// be sanitized and executed by the Docker daemon.
func validateComposeSecurity(doc map[string]any) error {
	services, ok := doc["services"].(map[string]any)
	if !ok || len(services) == 0 {
		return fmt.Errorf("compose config does not define services")
	}

	for name, rawService := range services {
		service, ok := rawService.(map[string]any)
		if !ok {
			continue
		}
		if valueIsTrue(service["privileged"]) {
			return composeSecurityError(name, "privileged containers are not allowed")
		}
		if valueIsTrue(service["use_api_socket"]) {
			return composeSecurityError(name, "use_api_socket is not allowed")
		}
		if composeValuePresent(service["provider"]) {
			return composeSecurityError(name, "provider services are not allowed because they execute host-side provider binaries")
		}
		if runtime := strings.ToLower(strings.TrimSpace(fmt.Sprint(service["runtime"]))); runtime != "" && runtime != "<nil>" && runtime != "runc" {
			return composeSecurityError(name, fmt.Sprintf("custom runtime %q is not allowed", runtime))
		}
		for _, field := range []string{"network_mode", "pid", "ipc", "uts", "userns_mode", "cgroup"} {
			if dangerousNamespaceMode(service[field]) {
				return composeSecurityError(name, fmt.Sprintf("%s=%q is not allowed", field, fmt.Sprint(service[field])))
			}
		}
		for _, field := range []string{
			"devices",
			"device_cgroup_rules",
			"cap_add",
			"volumes_from",
			"external_links",
			"gpus",
			"cgroup_parent",
			"credential_spec",
		} {
			if composeValuePresent(service[field]) {
				return composeSecurityError(name, field+" is not allowed")
			}
		}
		if securityOptsWeakenIsolation(service["security_opt"]) {
			return composeSecurityError(name, "security_opt may only enable no-new-privileges")
		}
		if lifecycleHasPrivilegedHook(service["pre_start"]) || lifecycleHasPrivilegedHook(service["post_start"]) || lifecycleHasPrivilegedHook(service["pre_stop"]) {
			return composeSecurityError(name, "privileged lifecycle hooks are not allowed")
		}
		if serviceHasBindMount(service["volumes"]) {
			return composeSecurityError(name, "host bind mounts are not allowed; use named volumes or bake files into the image")
		}
		if err := validateBuildSecurity(name, service["build"]); err != nil {
			return err
		}
	}

	if err := validateTopLevelVolumes(doc["volumes"]); err != nil {
		return err
	}
	if err := validateTopLevelNetworks(doc["networks"]); err != nil {
		return err
	}
	if composeValuePresent(doc["configs"]) || composeValuePresent(doc["secrets"]) {
		return fmt.Errorf("compose security policy: configs/secrets are not allowed; inject configuration through environment variables")
	}
	return nil
}

func composeSecurityError(service, message string) error {
	return fmt.Errorf("compose security policy: service %q: %s", service, message)
}

func dangerousNamespaceMode(value any) bool {
	mode := strings.ToLower(strings.TrimSpace(fmt.Sprint(value)))
	return mode == "host" || strings.HasPrefix(mode, "container:")
}

func valueIsTrue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func composeValuePresent(value any) bool {
	if value == nil {
		return false
	}
	switch typed := value.(type) {
	case string:
		value := strings.TrimSpace(typed)
		return value != "" && value != "<nil>"
	case []any:
		return len(typed) > 0
	case []string:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	case bool:
		return typed
	default:
		return true
	}
}

func securityOptsWeakenIsolation(value any) bool {
	if value == nil {
		return false
	}
	var options []string
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			options = append(options, fmt.Sprint(item))
		}
	case []string:
		options = typed
	case string:
		if strings.TrimSpace(typed) != "" {
			options = []string{typed}
		}
	default:
		return composeValuePresent(value)
	}
	for _, option := range options {
		normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(option), "=", ":"))
		if normalized != "no-new-privileges" && normalized != "no-new-privileges:true" {
			return true
		}
	}
	return false
}

func lifecycleHasPrivilegedHook(value any) bool {
	items, ok := value.([]any)
	if !ok {
		return false
	}
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if ok && valueIsTrue(item["privileged"]) {
			return true
		}
	}
	return false
}

func validateBuildSecurity(serviceName string, value any) error {
	build, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	if valueIsTrue(build["privileged"]) {
		return composeSecurityError(serviceName, "privileged image builds are not allowed")
	}
	if strings.EqualFold(strings.TrimSpace(fmt.Sprint(build["network"])), "host") {
		return composeSecurityError(serviceName, "build network=host is not allowed")
	}
	for _, field := range []string{"entitlements", "additional_contexts", "ssh", "secrets"} {
		if composeValuePresent(build[field]) {
			return composeSecurityError(serviceName, "build "+field+" is not allowed")
		}
	}
	return nil
}

func serviceHasBindMount(value any) bool {
	items, ok := value.([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		switch mount := item.(type) {
		case map[string]any:
			mountType := strings.ToLower(strings.TrimSpace(fmt.Sprint(mount["type"])))
			if mountType == "bind" {
				return true
			}
			// Be conservative when Compose emits a source path without a type.
			if mountType == "" && looksLikeHostPath(fmt.Sprint(mount["source"])) {
				return true
			}
		case string:
			parts := strings.SplitN(mount, ":", 2)
			if len(parts) == 2 && looksLikeHostPath(parts[0]) {
				return true
			}
		}
	}
	return false
}

func looksLikeHostPath(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "/") || strings.HasPrefix(value, ".") || strings.HasPrefix(value, "~") || strings.Contains(value, "/") || strings.Contains(value, "\\")
}

func validateTopLevelVolumes(value any) error {
	volumes, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	for name, raw := range volumes {
		spec, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if composeValuePresent(spec["external"]) {
			return fmt.Errorf("compose security policy: volume %q must not be external", name)
		}
		driver := strings.ToLower(strings.TrimSpace(fmt.Sprint(spec["driver"])))
		if driver != "" && driver != "local" && driver != "<nil>" {
			return fmt.Errorf("compose security policy: volume %q uses unsupported driver %q", name, driver)
		}
		if composeValuePresent(spec["driver_opts"]) {
			return fmt.Errorf("compose security policy: volume %q driver_opts are not allowed", name)
		}
	}
	return nil
}

func validateTopLevelNetworks(value any) error {
	networks, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	for name, raw := range networks {
		spec, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if composeValuePresent(spec["external"]) {
			return fmt.Errorf("compose security policy: network %q must not be external", name)
		}
		driver := strings.ToLower(strings.TrimSpace(fmt.Sprint(spec["driver"])))
		if driver != "" && driver != "bridge" && driver != "<nil>" {
			return fmt.Errorf("compose security policy: network %q uses unsupported driver %q", name, driver)
		}
		if composeValuePresent(spec["driver_opts"]) {
			return fmt.Errorf("compose security policy: network %q driver_opts are not allowed", name)
		}
	}
	return nil
}
