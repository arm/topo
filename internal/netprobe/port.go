package netprobe

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
)

var procNetTcpFiles = []string{"/proc/net/tcp", "/proc/net/tcp6"}

const (
	loopbackV4Hex     = "0100007F"
	loopbackV6Hex     = "00000000000000000000000001000000"
	tcpListeningState = "0A"
)

type parsedTable struct {
	columnIndexes map[string]int
	rows          [][]string
}

// IsRemotePortListening reports whether the target is listening beyond the local loopback on the given TCP port
func IsRemotePortListening(ctx context.Context, targetDest ssh.Destination, port string) (bool, error) {
	sshRunner := runner.NewSSH(targetDest)
	for _, path := range procNetTcpFiles {
		stdout, _, err := sshRunner.Run(ctx, fmt.Sprintf("cat %s", path))
		if err != nil {
			return false, err
		}

		listening, err := IsRemotePortListeningInProcNetTCP(stdout, port)
		if err != nil {
			return false, err
		}
		if listening {
			return true, nil
		}
	}
	return false, nil
}

func IsRemotePortListeningInProcNetTCP(procNetTCP, port string) (bool, error) {
	portNumber, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return false, fmt.Errorf("invalid TCP port %q: %w", port, err)
	}
	portHex := fmt.Sprintf("%04X", portNumber)

	table, err := parseTable(procNetTCP)
	if err != nil {
		return false, fmt.Errorf("parse proc net TCP table: %w", err)
	}
	localAddressColumn, ok := table.columnIndexes["local_address"]
	if !ok {
		return false, fmt.Errorf(`parse proc net TCP table: missing "local_address" column`)
	}
	stateColumn, ok := table.columnIndexes["st"]
	if !ok {
		return false, fmt.Errorf(`parse proc net TCP table: missing "st" column`)
	}

	for _, row := range table.rows {
		localAddress := row[localAddressColumn]
		state := row[stateColumn]
		if hasPort(localAddress, portHex) && state == tcpListeningState && !isLoopback(localAddress) {
			return true, nil
		}
	}
	return false, nil
}

func parseTable(rawTable string) (parsedTable, error) {
	lines := strings.Split(rawTable, "\n")
	headers := strings.Fields(lines[0])
	if len(headers) == 0 {
		return parsedTable{}, fmt.Errorf("table header is empty")
	}

	table := parsedTable{
		columnIndexes: make(map[string]int, len(headers)),
		rows:          make([][]string, 0, len(lines)-1),
	}
	for index, header := range headers {
		table.columnIndexes[header] = index
	}

	for index, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) < len(headers) {
			return parsedTable{}, fmt.Errorf(
				"row %d has %d fields, expected at least %d",
				index+2,
				len(fields),
				len(headers),
			)
		}
		table.rows = append(table.rows, fields)
	}
	return table, nil
}

func hasPort(localAddress, portHex string) bool {
	return strings.HasSuffix(localAddress, ":"+portHex)
}

func isLoopback(localAddress string) bool {
	return strings.HasPrefix(localAddress, loopbackV4Hex+":") || strings.HasPrefix(localAddress, loopbackV6Hex+":")
}
