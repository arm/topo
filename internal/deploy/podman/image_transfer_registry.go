package podman

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/arm/topo/internal/compose"
)

func TransferImagesViaRegistry(ctx context.Context, output io.Writer, sourceSocket, targetSocket Socket, composeFile, port string) error {
	images, err := compose.PodmanImageNames(composeFile)
	if err != nil {
		return err
	}

	for _, image := range images {
		if err := transferImageViaRegistry(ctx, output, sourceSocket, targetSocket, port, image); err != nil {
			return err
		}
	}
	return nil
}

func transferImageViaRegistry(ctx context.Context, output io.Writer, sourceSocket, targetSocket Socket, port, image string) error {
	registryTag := fmt.Sprintf("localhost:%s/%s", port, image)
	if err := tagImageForRegistry(ctx, output, sourceSocket, image, registryTag); err != nil {
		return err
	}

	digestReference, err := pushImageToRegistry(ctx, output, sourceSocket, registryTag)
	if err != nil {
		return err
	}

	if err := pullImageByDigest(ctx, output, targetSocket, digestReference); err != nil {
		return err
	}

	return restoreOriginalImageTag(ctx, output, targetSocket, digestReference, image)
}

func tagImageForRegistry(ctx context.Context, output io.Writer, sourceSocket Socket, image, registryTag string) error {
	return RunCommand(ctx, output, sourceSocket, "tag", image, registryTag)
}

func pushImageToRegistry(ctx context.Context, output io.Writer, sourceSocket Socket, registryTag string) (digestReference string, pushErr error) {
	digestFile, err := os.CreateTemp("", "topo-registry-digest-*")
	if err != nil {
		return "", fmt.Errorf("create registry digest file: %w", err)
	}
	digestFilePath := digestFile.Name()
	if err := digestFile.Close(); err != nil {
		return "", fmt.Errorf("close registry digest file: %w", err)
	}
	defer func() {
		// #nosec G703 -- digestFilePath is returned by os.CreateTemp.
		if err := os.Remove(digestFilePath); err != nil {
			pushErr = errors.Join(pushErr, fmt.Errorf("remove registry digest file: %w", err))
		}
	}()

	if err := RunCommand(ctx, output, sourceSocket, "push", "--tls-verify=false", "--digestfile", digestFilePath, registryTag); err != nil {
		return "", err
	}
	// #nosec G703 -- digestFilePath is returned by os.CreateTemp.
	digestBytes, err := os.ReadFile(digestFilePath)
	if err != nil {
		return "", fmt.Errorf("read registry digest file: %w", err)
	}
	digest := strings.TrimSpace(string(digestBytes))
	if digest == "" {
		return "", fmt.Errorf("registry push did not write an image digest")
	}
	return fmt.Sprintf("%s@%s", registryTag, digest), nil
}

func pullImageByDigest(ctx context.Context, output io.Writer, targetSocket Socket, digestReference string) error {
	return RunCommand(ctx, output, targetSocket, "pull", "--tls-verify=false", digestReference)
}

func restoreOriginalImageTag(ctx context.Context, output io.Writer, targetSocket Socket, digestReference, image string) error {
	return RunCommand(ctx, output, targetSocket, "tag", digestReference, image)
}
