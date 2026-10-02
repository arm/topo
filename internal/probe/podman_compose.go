package probe

import (
	"context"
	"strings"

	"github.com/arm/topo/internal/deploy/podman"
)

func CheckPodmanComposeIsDockerCompose(ctx context.Context) bool {
	cmd := podman.ComposeRawCommand(ctx, "version")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.HasPrefix(string(output), "Docker Compose version ")
}
