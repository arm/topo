package operation

import (
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"

	"github.com/arm/topo/internal/deploy/docker/command"
	"golang.org/x/sync/errgroup"
)

type DockerComposePipeTransfer struct {
	composeFile string
	source      command.Host
	dest        command.Host
}

func NewDockerComposePipeTransfer(composeFile string, source, dest command.Host) *DockerComposePipeTransfer {
	return &DockerComposePipeTransfer{
		composeFile: composeFile,
		source:      source,
		dest:        dest,
	}
}

func (t *DockerComposePipeTransfer) Description() string {
	return "Transfer images"
}

func (t *DockerComposePipeTransfer) Run(cmdOutput io.Writer) error {
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

func (t *DockerComposePipeTransfer) buildTransferCommands(imageName string) (*exec.Cmd, *exec.Cmd) {
	saveCmd := command.Docker(t.source, "save", imageName)
	loadCmd := command.Docker(t.dest, "load")
	return saveCmd, loadCmd
}

func (t *DockerComposePipeTransfer) getImagesFromCompose(cmdOutput io.Writer) ([]string, error) {
	cmd := command.DockerCompose(t.source, t.composeFile, "config", "--images")
	cmd.Stderr = cmdOutput
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get image names from compose file: %w", err)
	}
	var imageNames []string
	lines := strings.SplitSeq(strings.TrimSpace(string(output)), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			imageNames = append(imageNames, line)
		}
	}
	sort.Strings(imageNames)
	return imageNames, nil
}

func (t *DockerComposePipeTransfer) transferImage(cmdOutput io.Writer, imageName string) error {
	pipeReader, pipeWriter := io.Pipe()

	saveCmd, loadCmd := t.buildTransferCommands(imageName)
	saveCmd.Stdout = pipeWriter
	saveCmd.Stderr = cmdOutput
	loadCmd.Stdin = pipeReader
	loadCmd.Stdout = cmdOutput
	loadCmd.Stderr = cmdOutput

	var g errgroup.Group
	g.Go(func() error {
		defer pipeWriter.Close() //nolint:errcheck
		if err := saveCmd.Run(); err != nil {
			return fmt.Errorf("failed to save image: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		defer pipeReader.Close() //nolint:errcheck
		if err := loadCmd.Run(); err != nil {
			return fmt.Errorf("failed to load image: %w", err)
		}
		return nil
	})

	return g.Wait()
}
