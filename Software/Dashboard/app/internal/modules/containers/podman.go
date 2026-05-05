package containers

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// Container is a slim view of `podman ps -a --format=json` for the dashboard.
type Container struct {
	ID      string
	Name    string
	Image   string
	State   string
	Status  string
	Ports   string
	Created time.Time
}

// rawContainer mirrors the fields we care about from podman's JSON output.
// Names is an array; Ports is an array of objects.
type rawContainer struct {
	ID        string   `json:"Id"`
	Names     []string `json:"Names"`
	Image     string   `json:"Image"`
	State     string   `json:"State"`
	Status    string   `json:"Status"`
	StartedAt int64    `json:"StartedAt"`
	CreatedAt int64    `json:"Created"`
	Ports     []struct {
		HostIP        string `json:"host_ip"`
		ContainerPort int    `json:"container_port"`
		HostPort      int    `json:"host_port"`
		Protocol      string `json:"protocol"`
	} `json:"Ports"`
}

func List(ctx context.Context) ([]Container, error) {
	cmd := exec.CommandContext(ctx, "podman", "ps", "-a", "--format=json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("podman ps: %w", err)
	}
	var raw []rawContainer
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("decode podman output: %w", err)
	}
	containers := make([]Container, 0, len(raw))
	for _, r := range raw {
		name := ""
		if len(r.Names) > 0 {
			name = r.Names[0]
		}
		containers = append(containers, Container{
			ID:      shortID(r.ID),
			Name:    name,
			Image:   r.Image,
			State:   r.State,
			Status:  r.Status,
			Ports:   formatPorts(r.Ports),
			Created: time.Unix(r.CreatedAt, 0),
		})
	}
	return containers, nil
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func formatPorts(ports []struct {
	HostIP        string `json:"host_ip"`
	ContainerPort int    `json:"container_port"`
	HostPort      int    `json:"host_port"`
	Protocol      string `json:"protocol"`
}) string {
	if len(ports) == 0 {
		return ""
	}
	out := ""
	for i, p := range ports {
		if i > 0 {
			out += ", "
		}
		if p.HostPort != 0 {
			out += fmt.Sprintf("%d→%d/%s", p.HostPort, p.ContainerPort, p.Protocol)
		} else {
			out += fmt.Sprintf("%d/%s", p.ContainerPort, p.Protocol)
		}
	}
	return out
}
