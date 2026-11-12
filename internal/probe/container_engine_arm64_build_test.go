package probe_test

import (
	"testing"

	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/runner"
	"github.com/stretchr/testify/assert"
)

func TestCheckDockerARM64BuildUnavailable(t *testing.T) {
	for _, scenario := range []struct{ name, output, message string }{
		{"inactive builder", "Status: inactive\nPlatforms: linux/arm64", "inactive"},
		{"inspection error", "Error: unreachable\nPlatforms: linux/arm64", "unreachable"},
		{"missing platforms", "Status: running", "capabilities are unavailable"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			r := &runner.Fake{Commands: map[string]runner.FakeResult{
				"docker buildx inspect": {Output: scenario.output},
			}}

			err := probe.CheckDockerARM64Build(t.Context(), r)

			assert.ErrorContains(t, err, scenario.message)
			assert.NotErrorIs(t, err, probe.ErrARM64BuildUnsupported)
		})
	}
}

func TestCheckPodmanARM64BuildReportedEmulation(t *testing.T) {
	r := &runner.Fake{Commands: map[string]runner.FakeResult{
		"podman info --format json": {Output: `{"host":{"arch":"amd64","os":"linux","serviceIsRemote":true,"emulatedArchitectures":["linux/arm64"]}}`},
	}}

	err := probe.CheckPodmanARM64Build(t.Context(), r)

	assert.NoError(t, err)
}

func TestCheckPodmanARM64BuildMachineFallback(t *testing.T) {
	t.Setenv("CONTAINER_CONNECTION", "")
	t.Setenv("CONTAINER_HOST", "")

	r := &runner.Fake{Commands: map[string]runner.FakeResult{
		"podman info --format json": {Output: `{"host":{"arch":"amd64","os":"linux","serviceIsRemote":true}}`},
		"podman system connection list --format json": {Output: `[
			{"Name":"unrelated"},
			{"Name":"selected-root","URI":"ssh://root@127.0.0.1:2222/run/podman/podman.sock","Identity":"key","Default":true}
		]`},
		"podman machine list --format json": {Output: `[
			{"Name":"build-vm","Port":2222,"IdentityPath":"key"}
		]`},
		"podman machine ssh build-vm cat /proc/sys/fs/binfmt_misc/status":       {Output: "enabled\n"},
		"podman machine ssh build-vm cat /proc/sys/fs/binfmt_misc/qemu-aarch64": {Output: "enabled\nflags: F\n"},
	}}

	err := probe.CheckPodmanARM64Build(t.Context(), r)

	assert.NoError(t, err)
}
