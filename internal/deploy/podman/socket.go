package podman

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/arm/topo/internal/ssh"
)

type Socket struct {
	url                string
	localComposeSocket *localComposeSocket
}

type localComposeSocket struct {
	mutex sync.Mutex
	url   string
}

var LocalSocket = Socket{localComposeSocket: &localComposeSocket{}}

func NewSocket(url string) Socket {
	return Socket{url: url}
}

func (socket Socket) ConfigurePodmanEnv(environment []string) []string {
	if socket.url == "" {
		return environment
	}
	return append(environment, "CONTAINER_HOST="+socket.url)
}

func (socket Socket) ConfigureComposeEnv(ctx context.Context, environment []string) ([]string, error) {
	socketURL, err := socket.getComposeSocketURL(ctx)
	if err != nil {
		return nil, err
	}
	return append(environment, "DOCKER_HOST="+socketURL), nil
}

func (socket Socket) getComposeSocketURL(ctx context.Context) (string, error) {
	if socket.localComposeSocket == nil {
		return socket.url, nil
	}
	return socket.localComposeSocket.getURL(ctx)
}

func (socket *localComposeSocket) getURL(ctx context.Context) (string, error) {
	socket.mutex.Lock()
	defer socket.mutex.Unlock()

	if socket.url == "" {
		url, err := ResolveLocalComposeSocket(ctx)
		if err != nil {
			return "", err
		}
		socket.url = url
	}
	return socket.url, nil
}

type podmanConnection struct {
	Name      string
	URI       string
	Default   bool
	IsMachine bool
}

type machineConnectionInfo struct {
	PodmanSocket *machineSocket
	PodmanPipe   *machinePipe
}

type machineSocket struct {
	Path string
}

type machinePipe struct {
	Path string
}

// ResolveLocalComposeSocket returns the host-accessible Podman API endpoint
// that the external Compose provider must use for a local deployment.
func ResolveLocalComposeSocket(ctx context.Context) (string, error) {
	if runtime.GOOS == "darwin" {
		return resolveDarwinComposeSocket(ctx)
	}
	if runtime.GOOS == "windows" {
		return resolveWindowsComposeSocket(ctx)
	}
	return resolveNativeComposeSocket(ctx)
}

// ResolveRemoteSocketPath returns the absolute Unix socket path reported by Podman
// on target.
func ResolveRemoteSocketPath(ctx context.Context, target ssh.Destination) (string, error) {
	output, _, err := ssh.RunCommand(ctx, target, "podman info --format '{{.Host.RemoteSocket.Path}}'", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get remote Podman socket path: %w", err)
	}

	socketPath := strings.TrimPrefix(strings.TrimSpace(output), "unix://")
	if !path.IsAbs(socketPath) {
		return "", fmt.Errorf("remote Podman reported a non-local API socket path %q", socketPath)
	}
	return socketPath, nil
}

func resolveNativeComposeSocket(ctx context.Context) (string, error) {
	output, err := exec.CommandContext(ctx, "podman", "info", "--format", "{{.Host.RemoteSocket.Path}}").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Podman API socket: %w", err)
	}

	socketURL, err := unixSocketURL(strings.TrimSpace(string(output)))
	if err != nil {
		return "", err
	}
	if err := requireSocket(socketURL); err != nil {
		return "", err
	}
	return socketURL, nil
}

func resolveDarwinComposeSocket(ctx context.Context) (string, error) {
	connection, err := defaultPodmanConnection(ctx)
	if err != nil {
		return "", err
	}
	if !connection.IsMachine {
		if isDarwinComposeEndpoint(connection.URI) {
			return connection.URI, nil
		}
		return "", fmt.Errorf("default Podman connection %q has no Compose-compatible endpoint", connection.Name)
	}

	info, err := inspectPodmanMachine(ctx, connection)
	if err != nil {
		return "", err
	}
	if info.PodmanSocket == nil {
		return "", fmt.Errorf("podman machine %q has no host-accessible API socket", connection.Name)
	}
	socketURL, err := unixSocketURL(info.PodmanSocket.Path)
	if err != nil {
		return "", err
	}
	return socketURL, nil
}

func resolveWindowsComposeSocket(ctx context.Context) (string, error) {
	connection, err := defaultPodmanConnection(ctx)
	if err != nil {
		return "", err
	}
	if !connection.IsMachine {
		if isWindowsComposeEndpoint(connection.URI) {
			return connection.URI, nil
		}
		return "", fmt.Errorf("default Podman connection %q has no Compose-compatible endpoint", connection.Name)
	}

	info, err := inspectPodmanMachine(ctx, connection)
	if err != nil {
		return "", err
	}
	if info.PodmanPipe == nil {
		return "", fmt.Errorf("podman machine %q has no host-accessible API pipe", connection.Name)
	}
	pipeURL, err := namedPipeURL(info.PodmanPipe.Path)
	if err != nil {
		return "", err
	}
	return pipeURL, nil
}

func inspectPodmanMachine(ctx context.Context, connection podmanConnection) (machineConnectionInfo, error) {
	output, err := exec.CommandContext(ctx, "podman", "machine", "inspect", connection.Name, "--format", "{{json .ConnectionInfo}}").Output()
	if err != nil {
		return machineConnectionInfo{}, fmt.Errorf("failed to inspect Podman machine %q: %w", connection.Name, err)
	}

	var info machineConnectionInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return machineConnectionInfo{}, fmt.Errorf("failed to parse Podman machine connection information: %w", err)
	}
	return info, nil
}

func defaultPodmanConnection(ctx context.Context) (podmanConnection, error) {
	output, err := exec.CommandContext(ctx, "podman", "system", "connection", "list", "--format", "json").Output()
	if err != nil {
		return podmanConnection{}, fmt.Errorf("failed to list Podman connections: %w", err)
	}

	var connections []podmanConnection
	if err := json.Unmarshal(output, &connections); err != nil {
		return podmanConnection{}, fmt.Errorf("failed to parse Podman connections: %w", err)
	}
	for _, connection := range connections {
		if connection.Default {
			return connection, nil
		}
	}
	return podmanConnection{}, fmt.Errorf("no default Podman connection is configured")
}

func isDarwinComposeEndpoint(endpoint string) bool {
	return strings.HasPrefix(endpoint, "unix://") || strings.HasPrefix(endpoint, "tcp://")
}

func isWindowsComposeEndpoint(endpoint string) bool {
	return strings.HasPrefix(endpoint, "npipe://") || strings.HasPrefix(endpoint, "tcp://")
}

func unixSocketURL(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("podman did not report an API socket")
	}
	if strings.HasPrefix(path, "unix://") {
		return path, nil
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("podman reported a non-local API socket path %q", path)
	}
	return "unix://" + path, nil
}

func namedPipeURL(path string) (string, error) {
	path = strings.ReplaceAll(path, `\`, "/")
	if !strings.HasPrefix(path, "//./pipe/") {
		return "", fmt.Errorf("podman reported an invalid named pipe path %q", path)
	}
	return "npipe:////./pipe/" + strings.TrimPrefix(path, "//./pipe/"), nil
}

func requireSocket(socketURL string) error {
	parsedURL, err := url.Parse(socketURL)
	if err != nil {
		return fmt.Errorf("failed to parse Podman API socket URL: %w", err)
	}
	if parsedURL.Scheme != "unix" {
		return fmt.Errorf("podman reported an unsupported API socket URL %q", socketURL)
	}
	if _, err := os.Stat(parsedURL.Path); err != nil { // #nosec G703 -- socket URL comes from Podman.
		return fmt.Errorf("podman API socket %s is unavailable; start podman.socket: %w", socketURL, err)
	}
	return nil
}
