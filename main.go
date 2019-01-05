package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/containers/image/image"
	"github.com/containers/image/pkg/blobinfocache"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/pkg/errors"
)

func main() {
	err := ConvertImage("docker-daemon:lambda-php:latest")
	if err != nil {
		fmt.Printf("Error: %+v", err)
		os.Exit(1)
	}
}

func ConvertImage(name string) (retErr error) {
	// Get image's layer data from image name
	ref, err := alltransports.ParseImageName(name)
	if err != nil {
		return err
	}

	sys := &types.SystemContext{}

	ctx := context.Background()

	cache := blobinfocache.DefaultCache(sys)

	rawSource, err := ref.NewImageSource(ctx, sys)
	if err != nil {
		return err
	}

	src, err := image.FromSource(ctx, sys, rawSource)
	if err != nil {
		if closeErr := rawSource.Close(); closeErr != nil {
			return errors.Wrapf(err, " (close error: %v)", closeErr)
		}

		return err
	}
	defer func() {
		if err := src.Close(); err != nil {
			retErr = errors.Wrapf(retErr, " (close error: %v)", err)
		}
	}()

	layerInfos := src.LayerInfos()

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// Unpack each layer onto disk
	for _, layerInfo := range layerInfos {
		blobStream, _, err := rawSource.GetBlob(ctx, layerInfo, cache)
		if err != nil {
			return err
		}

		imageDir := filepath.Join(dir, "image-output", name, string(layerInfo.Digest))

		if err := os.MkdirAll(imageDir, 0777); err != nil {
			return err
		}

		fmt.Printf("Layer %s, size %d, media type %s\n", layerInfo.Digest, layerInfo.Size, layerInfo.MediaType)
		fmt.Printf("Dir %s\n", imageDir)

		tarCmd := exec.Command("tar", "-p", "-x", "-C", imageDir)
		tarCmd.Stdin = blobStream
		if output, err := tarCmd.CombinedOutput(); err != nil {
			fmt.Printf("combined out:\n%s\n", string(output))
			return err
		}

		blobStream.Close()
	}

	return nil
}
