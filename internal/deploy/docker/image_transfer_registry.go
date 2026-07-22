package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/deploy/command"
)

var digestRegexp = regexp.MustCompile(`digest: (sha256:[a-f0-9]+)`)

func TransferImagesViaRegistry(ctx context.Context, output io.Writer, composeFile string, source, destination command.Host, port string) error {
	images, err := compose.ImageNames(composeFile)
	if err != nil {
		return err
	}

	for _, image := range images {
		if err := transferImageViaRegistry(ctx, source, destination, port, output, image); err != nil {
			return err
		}
	}
	return nil
}

func transferImageViaRegistry(ctx context.Context, source, destination command.Host, port string, output io.Writer, image string) error {
	registryTag := fmt.Sprintf("localhost:%s/%s", port, image)
	if err := tagImageForRegistry(ctx, source, image, registryTag, output); err != nil {
		return err
	}

	digestReference, err := pushImageToRegistry(ctx, source, registryTag, output)
	if err != nil {
		return err
	}

	if err := pullImageByDigest(ctx, destination, digestReference, output); err != nil {
		return err
	}

	return restoreOriginalImageTag(ctx, destination, digestReference, image, output)
}

func tagImageForRegistry(ctx context.Context, source command.Host, image, registryTag string, output io.Writer) error {
	return command.RunDocker(ctx, source, output, "tag", image, registryTag)
}

func pushImageToRegistry(ctx context.Context, source command.Host, registryTag string, output io.Writer) (string, error) {
	pushCommand := command.DockerContext(ctx, source, "push", registryTag)
	var pushOutput bytes.Buffer
	pushCommand.Stdout = io.MultiWriter(output, &pushOutput)
	pushCommand.Stderr = output
	if err := pushCommand.Run(); err != nil {
		return "", fmt.Errorf("failed to execute %s: %w", strings.Join(pushCommand.Args, " "), err)
	}

	digest, err := ParseDigestFromPushOutput(pushOutput.String())
	if err != nil {
		return "", fmt.Errorf("failed to parse digest after pushing %s: %w", registryTag, err)
	}
	return fmt.Sprintf("%s@%s", registryTag, digest), nil
}

func pullImageByDigest(ctx context.Context, destination command.Host, digestReference string, output io.Writer) error {
	return command.RunDocker(ctx, destination, output, "pull", digestReference)
}

func restoreOriginalImageTag(ctx context.Context, destination command.Host, digestReference, image string, output io.Writer) error {
	return command.RunDocker(ctx, destination, output, "tag", digestReference, image)
}

func ParseDigestFromPushOutput(output string) (string, error) {
	match := digestRegexp.FindStringSubmatch(output)
	if match == nil {
		return "", fmt.Errorf("no digest found in push output")
	}
	return match[1], nil
}
