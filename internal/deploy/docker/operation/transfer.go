package operation

import (
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/command"
	"github.com/arm-debug/topo-cli/internal/ssh"
	"golang.org/x/sync/errgroup"
)

type Transfer struct {
	composeFile string
	sourceHost  ssh.Host
	targetHost  ssh.Host
}

func NewTransfer(composeFile string, sourceHost, targetHost ssh.Host) *Transfer {
	return &Transfer{
		composeFile: composeFile,
		sourceHost:  sourceHost,
		targetHost:  targetHost,
	}
}

func (t *Transfer) Description() string {
	return "Transfer images"
}

func (t *Transfer) Run(cmdOutput io.Writer) error {
	images, err := t.getImagesFromCompose(cmdOutput)
	if err != nil {
		return err
	}
	var g errgroup.Group
	for _, image := range images {
		g.Go(func() error {
			return t.transferImage(cmdOutput, image)
		})
	}
	return g.Wait()
}

func (t *Transfer) DryRun(output io.Writer) error {
	images, err := t.getImagesFromCompose(io.Discard)
	if err != nil {
		return err
	}
	for _, image := range images {
		saveCmd, loadCmd := t.buildTransferCommands(image)
		fmt.Fprintf(output, "%s | %s\n", command.String(saveCmd), command.String(loadCmd))
	}
	return nil
}

func (t *Transfer) buildTransferCommands(imageName string) (*exec.Cmd, *exec.Cmd) {
	saveCmd := command.Docker(t.sourceHost, "save", imageName)
	loadCmd := command.Docker(t.targetHost, "load")
	return saveCmd, loadCmd
}

func (t *Transfer) getImagesFromCompose(cmdOutput io.Writer) ([]string, error) {
	cmd := command.DockerCompose(t.sourceHost, t.composeFile, "config", "--images")
	cmd.Stderr = cmdOutput
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get image names from compose file: %w", err)
	}
	var imageNames []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			imageNames = append(imageNames, line)
		}
	}
	sort.Strings(imageNames)
	return imageNames, nil
}

func (t *Transfer) transferImage(cmdOutput io.Writer, imageName string) error {
	pipeReader, pipeWriter := io.Pipe()

	saveCmd, loadCmd := t.buildTransferCommands(imageName)
	saveCmd.Stdout = pipeWriter
	saveCmd.Stderr = cmdOutput
	loadCmd.Stdin = pipeReader
	loadCmd.Stdout = cmdOutput
	loadCmd.Stderr = cmdOutput

	var g errgroup.Group
	g.Go(func() error {
		defer pipeWriter.Close()
		if err := saveCmd.Run(); err != nil {
			return fmt.Errorf("failed to save image: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		defer pipeReader.Close()
		if err := loadCmd.Run(); err != nil {
			return fmt.Errorf("failed to load image: %w", err)
		}
		return nil
	})

	return g.Wait()
}
