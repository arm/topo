package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/arm/topo/internal/compose"
)

var digestRegexp = regexp.MustCompile(`digest: (sha256:[a-f0-9]+)`)

func TransferImagesViaRegistry(ctx context.Context, output io.Writer, source, destination Host, composeFile, port string) error {
	images, err := compose.ImageNames(composeFile)
	if err != nil {
		return err
	}

	for _, image := range images {
		if err := transferImageViaRegistry(ctx, output, source, destination, port, image); err != nil {
			return err
		}
	}
	return nil
}

func transferImageViaRegistry(ctx context.Context, output io.Writer, source, destination Host, port, image string) error {
	registryTag := fmt.Sprintf("localhost:%s/%s", port, image)
	if err := tagImageForRegistry(ctx, output, source, image, registryTag); err != nil {
		return err
	}

	digestReference, err := pushImageToRegistry(ctx, output, source, registryTag)
	if err != nil {
		return err
	}

	if err := pullImageByDigest(ctx, output, destination, digestReference); err != nil {
		return err
	}

	return restoreOriginalImageTag(ctx, output, destination, digestReference, image)
}

func tagImageForRegistry(ctx context.Context, output io.Writer, source Host, image, registryTag string) error {
	return RunCommand(ctx, output, source, "tag", image, registryTag)
}

func pushImageToRegistry(ctx context.Context, output io.Writer, source Host, registryTag string) (string, error) {
	pushCommand := Command(ctx, source, "push", registryTag)
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

func pullImageByDigest(ctx context.Context, output io.Writer, destination Host, digestReference string) error {
	return RunCommand(ctx, output, destination, "pull", digestReference)
}

func restoreOriginalImageTag(ctx context.Context, output io.Writer, destination Host, digestReference, image string) error {
	return RunCommand(ctx, output, destination, "tag", digestReference, image)
}

func ParseDigestFromPushOutput(output string) (string, error) {
	match := digestRegexp.FindStringSubmatch(output)
	if match == nil {
		return "", fmt.Errorf("no digest found in push output")
	}
	return match[1], nil
}
