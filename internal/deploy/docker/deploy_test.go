package docker_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/docker/operation"
	"github.com/arm/topo/internal/deploy/docker/testutil"
	goperation "github.com/arm/topo/internal/operation"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDeployment(t *testing.T) {
	composeFile := "compose.yaml"

	t.Run("includes transfer operation for remote host", func(t *testing.T) {
		remoteHost := ssh.Host("user@remote")
		deployOpts := docker.DeployOptions{TargetHost: remoteHost}
		got, _ := docker.NewDeployment(composeFile, deployOpts)

		want := goperation.Sequence{
			operation.NewDockerComposeBuild(composeFile, ssh.PlainLocalhost),
			operation.NewDockerComposePull(composeFile, ssh.PlainLocalhost),
			operation.NewDockerComposePipeTransfer(composeFile, ssh.PlainLocalhost, remoteHost),
			operation.NewDockerComposeRun(composeFile, remoteHost, operation.DockerComposeUpArgs{}),
		}
		assert.Equal(t, want, got)
	})

	t.Run("includes registry operations for remote host when enabled", func(t *testing.T) {
		remoteHost := ssh.Host("user@remote")
		port := operation.DefaultRegistryPort
		opts := docker.DeployOptions{TargetHost: remoteHost, WithRegistry: true, ForceRecreate: false, Port: port, UseSSHControlSockets: true}
		upArgs := operation.DockerComposeUpArgs{
			ForceRecreate: opts.ForceRecreate,
		}
		got, _ := docker.NewDeployment(composeFile, opts)

		want := goperation.Sequence{
			operation.NewDockerComposeBuild(composeFile, ssh.PlainLocalhost),
			operation.NewDockerComposePull(composeFile, ssh.PlainLocalhost),
		}
		want = append(want, operation.NewRunRegistry(port)...)
		want = append(want,
			ssh.NewSSHTunnelStart(remoteHost, port, opts.UseSSHControlSockets),
			operation.NewRegistryTransfer(composeFile, ssh.PlainLocalhost, remoteHost, port),
			ssh.NewSSHTunnelStop(remoteHost),
			operation.NewDockerComposeRun(composeFile, remoteHost, upArgs),
		)

		assert.Equal(t, want, got)
	})

	t.Run("excludes transfer operation for local host", func(t *testing.T) {
		tests := []struct {
			name string
			opts docker.DeployOptions
		}{
			{
				name: "default",
				opts: docker.DeployOptions{},
			},
			{
				name: "force recreate",
				opts: docker.DeployOptions{
					ForceRecreate: true,
				},
			},
			{
				name: "no recreate",
				opts: docker.DeployOptions{
					NoRecreate: true,
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				deployOpts := docker.DeployOptions{
					TargetHost:    ssh.PlainLocalhost,
					ForceRecreate: tt.opts.ForceRecreate,
					NoRecreate:    tt.opts.NoRecreate,
				}
				got, _ := docker.NewDeployment(composeFile, deployOpts)

				upArgs := operation.DockerComposeUpArgs{
					ForceRecreate: tt.opts.ForceRecreate,
					NoRecreate:    tt.opts.NoRecreate,
				}
				want := goperation.Sequence{
					operation.NewDockerComposeBuild(composeFile, ssh.PlainLocalhost),
					operation.NewDockerComposePull(composeFile, ssh.PlainLocalhost),
					operation.NewDockerComposeRun(composeFile, ssh.PlainLocalhost, upArgs),
				}

				assert.Equal(t, want, got)
			})
		}
	})

	t.Run("returns an SSH tunnel cleanup operation for remote host", func(t *testing.T) {
		remoteHost := ssh.Host("user@remote")
		deployOpts := docker.DeployOptions{TargetHost: remoteHost, WithRegistry: true, UseSSHControlSockets: true}
		_, cleanup := docker.NewDeployment(composeFile, deployOpts)

		want := ssh.NewSSHTunnelStop(remoteHost)
		assert.Equal(t, want, cleanup)
	})

	t.Run("does not return an SSH tunnel cleanup operation for local host", func(t *testing.T) {
		localHost := ssh.PlainLocalhost
		deployOpts := docker.DeployOptions{TargetHost: localHost, WithRegistry: true}
		_, cleanup := docker.NewDeployment(composeFile, deployOpts)

		var want goperation.Operation = nil
		assert.Equal(t, want, cleanup)
	})

	t.Run("does not use SSH control sockets when disabled", func(t *testing.T) {
		remoteHost := ssh.Host("user@remote")
		port := operation.DefaultRegistryPort
		opts := docker.DeployOptions{TargetHost: remoteHost, WithRegistry: true, ForceRecreate: false, Port: port, UseSSHControlSockets: false}
		upArgs := operation.DockerComposeUpArgs{
			ForceRecreate: opts.ForceRecreate,
		}
		got, _ := docker.NewDeployment(composeFile, opts)

		wantTunnelStart, wantTunnelEnd := ssh.NewSSHTunnel(remoteHost, opts.Port, opts.UseSSHControlSockets)
		want := goperation.Sequence{
			operation.NewDockerComposeBuild(composeFile, ssh.PlainLocalhost),
			operation.NewDockerComposePull(composeFile, ssh.PlainLocalhost),
		}
		want = append(want, operation.NewRunRegistry(port)...)
		want = append(want,
			wantTunnelStart,
			operation.NewRegistryTransfer(composeFile, ssh.PlainLocalhost, remoteHost, port),
			wantTunnelEnd,
			operation.NewDockerComposeRun(composeFile, remoteHost, upArgs),
		)

		assert.Equal(t, want, got)
	})
}

func TestDeployment(t *testing.T) {
	testutil.RequireDocker(t)

	t.Run("Run", func(t *testing.T) {
		target := testutil.StartTargetContainer(t)

		t.Run("builds images, transfers them, and starts services", func(t *testing.T) {
			remoteDockerHost := ssh.Host(target.SSHConnectionString)
			tmpDir := t.TempDir()
			dockerFilePath := filepath.Join(tmpDir, "Dockerfile")
			dockerFileContent := `
FROM alpine:latest
CMD ["tail", "-f", "/dev/null"]
`
			testutil.RequireWriteFile(t, dockerFilePath, dockerFileContent)
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			composeFileContent := fmt.Sprintf(`
name: %s
services:
  busybox:
    image: busybox
    command: ["tail", "-f", "/dev/null"]
  a-service:
    build: .
`, testutil.TestProjectName(t))
			testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
			t.Cleanup(func() { testutil.ForceComposeDown(t, composeFilePath) })

			deployOpts := docker.DeployOptions{TargetHost: remoteDockerHost}
			d, _ := docker.NewDeployment(composeFilePath, deployOpts)

			err := d.Run(os.Stdout)

			require.NoError(t, err)
			testutil.AssertContainersRunning(t, remoteDockerHost, composeFilePath)
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("prints all commands", func(t *testing.T) {
			var buf bytes.Buffer
			tmpDir := t.TempDir()
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			composeFileContent := `
services:
  alpine:
    image: alpine:latest
  busybox:
    image: busybox
`
			testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
			deployOpts := docker.DeployOptions{TargetHost: ssh.Host("user@remote")}
			d, _ := docker.NewDeployment(composeFilePath, deployOpts)

			err := d.DryRun(&buf)

			require.NoError(t, err)
			got := buf.String()
			want := fmt.Sprintf(`
┌─ Build images ────────────────────────────────────────
docker compose -f %[1]s build

┌─ Pull images ─────────────────────────────────────────
docker compose -f %[1]s pull

┌─ Transfer images ─────────────────────────────────────
docker save alpine:latest | docker -H ssh://user@remote load
docker save busybox | docker -H ssh://user@remote load

┌─ Start services ──────────────────────────────────────
docker -H ssh://user@remote compose -f %[1]s up -d --no-build --pull never
`, composeFilePath)
			assert.Equal(t, want, got)
		})
	})
}
