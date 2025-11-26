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
	cmdOutput   io.Writer
	composeFile string
	sourceHost  ssh.Host
	targetHost  ssh.Host
}

func NewTransfer(cmdOutput io.Writer, composeFile string, sourceHost, targetHost ssh.Host) *Transfer {
	return &Transfer{
		cmdOutput:   cmdOutput,
		composeFile: composeFile,
		sourceHost:  sourceHost,
		targetHost:  targetHost,
	}
}

func (t *Transfer) Description() string {
	return "Transfer images"
}

func (t *Transfer) Run() error {
	images, err := t.getImagesFromCompose()
	if err != nil {
		return err
	}
	var g errgroup.Group
	for _, image := range images {
		g.Go(func() error {
			return t.transferImage(image)
		})
	}
	return g.Wait()
}

func (t *Transfer) DryRun(w io.Writer) error {
	images, err := t.getImagesFromCompose()
	if err != nil {
		return err
	}
	for _, image := range images {
		saveCmd, loadCmd := t.buildTransferCommands(image)
		fmt.Fprintf(w, "%s | %s\n", command.String(saveCmd), command.String(loadCmd))
	}
	return nil
}

func (t *Transfer) buildTransferCommands(imageName string) (*exec.Cmd, *exec.Cmd) {
	saveCmd := command.Docker(t.sourceHost, "save", imageName)
	loadCmd := command.Docker(t.targetHost, "load")
	return saveCmd, loadCmd
}

func (t *Transfer) getImagesFromCompose() ([]string, error) {
	cmd := command.DockerCompose(t.sourceHost, t.composeFile, "config", "--images")
	cmd.Stderr = t.cmdOutput
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

func (t *Transfer) transferImage(imageName string) error {
	pipeReader, pipeWriter := io.Pipe()

	saveCmd, loadCmd := t.buildTransferCommands(imageName)
	saveCmd.Stdout = pipeWriter
	saveCmd.Stderr = t.cmdOutput
	loadCmd.Stdin = pipeReader
	loadCmd.Stdout = t.cmdOutput
	loadCmd.Stderr = t.cmdOutput

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
