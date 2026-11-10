package probe

import (
	"testing"
	"testing/fstest"

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

			err := CheckDockerARM64Build(t.Context(), r)

			assert.ErrorContains(t, err, scenario.message)
			assert.NotErrorIs(t, err, ErrARM64BuildUnsupported)
		})
	}
}

func TestCheckPodmanARM64BuildReportedEmulation(t *testing.T) {
	r := &runner.Fake{Commands: map[string]runner.FakeResult{
		"podman info --format json": {Output: `{"host":{"arch":"amd64","os":"linux","serviceIsRemote":true,"emulatedArchitectures":["linux/arm64"]}}`},
	}}

	err := CheckPodmanARM64Build(t.Context(), r)

	assert.NoError(t, err)
}

func TestCheckPodmanARM64BuildUnavailable(t *testing.T) {
	binfmt := fstest.MapFS{
		"status":       {Data: []byte("enabled\n")},
		"qemu-aarch64": {Data: []byte("enabled\nflags: F\n")},
	}

	for _, scenario := range []struct{ name, output, message string }{
		{"missing host", `{"host":null}`, "did not report its host architecture and operating system"},
		{"remote cannot use local emulation", `{"host":{"arch":"amd64","os":"linux","serviceIsRemote":true}}`, "no configured Podman connection matches the selected engine"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			r := &runner.Fake{Commands: map[string]runner.FakeResult{
				"podman info --format json":                   {Output: scenario.output},
				"podman system connection list --format json": {Output: `[]`},
			}}

			err := checkPodmanARM64Build(t.Context(), r, binfmt)

			assert.ErrorContains(t, err, scenario.message)
			assert.NotErrorIs(t, err, ErrARM64BuildUnsupported)
		})
	}
}

func TestCheckPodmanARM64BuildBinfmtFallback(t *testing.T) {
	r := &runner.Fake{Commands: map[string]runner.FakeResult{
		"podman info --format json": {Output: `{"host":{"arch":"amd64","os":"linux","serviceIsRemote":false}}`},
	}}

	for _, scenario := range []struct {
		name, status, registration string
		wantErr                    error
	}{
		{"registered", "enabled", "enabled\nflags: F\n", nil},
		{"missing registration", "enabled", "", ErrARM64BuildUnsupported},
		{"globally disabled", "disabled", "enabled\nflags: F\n", ErrARM64BuildUnsupported},
		{"disabled emulator", "enabled", "disabled\nflags: F\n", ErrARM64BuildUnsupported},
		{"missing F flag", "enabled", "enabled\nflags: POC\n", ErrARM64BuildUnsupported},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			binfmt := fstest.MapFS{"status": {Data: []byte(scenario.status + "\n")}}
			if scenario.registration != "" {
				binfmt["qemu-aarch64"] = &fstest.MapFile{Data: []byte(scenario.registration)}
			}

			err := checkPodmanARM64Build(t.Context(), r, binfmt)

			assert.ErrorIs(t, err, scenario.wantErr)
		})
	}
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

	err := checkPodmanARM64Build(t.Context(), r, fstest.MapFS{})

	assert.NoError(t, err)
}
