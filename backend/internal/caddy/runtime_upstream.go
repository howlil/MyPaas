package caddy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const runtimeUpstreamMode = "runtime"

const (
	defaultProjectNetwork = "mypaas-projects"
	defaultRoutingNetwork = "mypaas-routing"
)

type runtimeInspectRow struct {
	ID              string `json:"Id"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
		Networks map[string]struct {
			Aliases []string `json:"Aliases"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

type runtimeRouteTarget struct {
	ContainerID         string
	ContainerPort       string
	RoutingAttached     bool
	RoutingAliasPresent bool
}

func runtimeRouteAlias(hostPort int32) string {
	return fmt.Sprintf("mypaas-port-%d", hostPort)
}

// upstreamDial keeps the existing fixed-host behavior for development and
// compatibility deployments. Production sets CADDY_UPSTREAM_HOST=runtime. In
// that mode the allocated host port is only a runtime lookup key: the selected
// container is attached to a dedicated routing network with an explicit,
// Docker/Podman-portable DNS alias and Caddy dials that alias directly.
func (c *Client) upstreamDial(ctx context.Context, hostPort int32) (string, error) {
	if strings.TrimSpace(c.upstreamHost) != runtimeUpstreamMode {
		return fmt.Sprintf("%s:%d", c.upstreamHost, hostPort), nil
	}

	projectNetwork := strings.TrimSpace(os.Getenv("PROJECT_NETWORK"))
	if projectNetwork == "" {
		projectNetwork = defaultProjectNetwork
	}
	routingNetwork := strings.TrimSpace(os.Getenv("ROUTING_NETWORK"))
	if routingNetwork == "" {
		routingNetwork = defaultRoutingNetwork
	}
	if routingNetwork == projectNetwork {
		return "", fmt.Errorf("ROUTING_NETWORK must be distinct from PROJECT_NETWORK")
	}

	idsRaw, err := exec.CommandContext(ctx, "docker", "ps", "-q").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("list running containers for Caddy upstream: %w: %s", err, strings.TrimSpace(string(idsRaw)))
	}
	ids := strings.Fields(string(idsRaw))
	if len(ids) == 0 {
		return "", fmt.Errorf("resolve Caddy upstream for host port %d: no running containers", hostPort)
	}

	args := append([]string{"inspect"}, ids...)
	inspectRaw, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("inspect running containers for Caddy upstream: %w: %s", err, strings.TrimSpace(string(inspectRaw)))
	}

	alias := runtimeRouteAlias(hostPort)
	target, err := runtimeTargetFromInspect(inspectRaw, projectNetwork, routingNetwork, alias, hostPort)
	if err != nil {
		return "", err
	}

	if target.RoutingAttached && !target.RoutingAliasPresent {
		out, disconnectErr := exec.CommandContext(
			ctx,
			"docker",
			"network",
			"disconnect",
			routingNetwork,
			target.ContainerID,
		).CombinedOutput()
		if disconnectErr != nil {
			return "", fmt.Errorf(
				"refresh routing alias for runtime %s on network %q: disconnect: %w: %s",
				strings.TrimSpace(target.ContainerID),
				routingNetwork,
				disconnectErr,
				strings.TrimSpace(string(out)),
			)
		}
		target.RoutingAttached = false
	}

	if !target.RoutingAttached {
		out, connectErr := exec.CommandContext(
			ctx,
			"docker",
			"network",
			"connect",
			"--alias",
			alias,
			routingNetwork,
			target.ContainerID,
		).CombinedOutput()
		if connectErr != nil {
			return "", fmt.Errorf(
				"attach runtime %s to routing network %q: %w: %s",
				strings.TrimSpace(target.ContainerID),
				routingNetwork,
				connectErr,
				strings.TrimSpace(string(out)),
			)
		}
	}

	return fmt.Sprintf("%s:%s", alias, target.ContainerPort), nil
}

func runtimeTargetFromInspect(raw []byte, projectNetwork, routingNetwork, alias string, hostPort int32) (runtimeRouteTarget, error) {
	var rows []runtimeInspectRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return runtimeRouteTarget{}, fmt.Errorf("decode runtime inspect for Caddy upstream: %w", err)
	}

	wantedHostPort := strconv.Itoa(int(hostPort))
	var candidateErr error
	for _, row := range rows {
		containerPort := ""
		for portSpec, bindings := range row.NetworkSettings.Ports {
			for _, binding := range bindings {
				if strings.TrimSpace(binding.HostPort) != wantedHostPort {
					continue
				}
				portText := strings.SplitN(strings.TrimSpace(portSpec), "/", 2)[0]
				parsed, err := strconv.Atoi(portText)
				if err != nil || parsed <= 0 || parsed > 65535 {
					candidateErr = fmt.Errorf("resolve Caddy upstream for host port %d: invalid container port %q", hostPort, portSpec)
					continue
				}
				containerPort = strconv.Itoa(parsed)
				break
			}
			if containerPort != "" {
				break
			}
		}
		if containerPort == "" {
			continue
		}

		if _, ok := row.NetworkSettings.Networks[projectNetwork]; !ok {
			if candidateErr == nil {
				candidateErr = fmt.Errorf("resolve Caddy upstream for host port %d: container %s is not attached to project network %q", hostPort, strings.TrimSpace(row.ID), projectNetwork)
			}
			continue
		}

		containerID := strings.TrimSpace(row.ID)
		if containerID == "" {
			if candidateErr == nil {
				candidateErr = fmt.Errorf("resolve Caddy upstream for host port %d: matched container has empty runtime ID", hostPort)
			}
			continue
		}

		routing, routingAttached := row.NetworkSettings.Networks[routingNetwork]
		return runtimeRouteTarget{
			ContainerID:         containerID,
			ContainerPort:       containerPort,
			RoutingAttached:     routingAttached,
			RoutingAliasPresent: routingAttached && stringSliceContains(routing.Aliases, alias),
		}, nil
	}

	if candidateErr != nil {
		return runtimeRouteTarget{}, candidateErr
	}
	return runtimeRouteTarget{}, fmt.Errorf("resolve Caddy upstream for host port %d: no running container owns that published port", hostPort)
}

func stringSliceContains(values []string, wanted string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == wanted {
			return true
		}
	}
	return false
}
