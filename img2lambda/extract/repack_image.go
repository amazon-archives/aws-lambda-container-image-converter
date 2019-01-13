package extract

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/awslabs/aws-lambda-container-image-converter/img2lambda/types"
	"github.com/containers/image/image"
	"github.com/containers/image/pkg/blobinfocache"
	"github.com/containers/image/transports/alltransports"
	imgtypes "github.com/containers/image/types"
	"github.com/pkg/errors"
)

// Converts container image to Lambda layer archive files
func RepackImage(imageName string, layerOutputDir string) (layers []types.LambdaLayer, retErr error) {
	log.Printf("Parsing the docker image %s", imageName)

	// Get image's layer data from image name
	ref, err := alltransports.ParseImageName(imageName)
	if err != nil {
		return nil, err
	}

	sys := &imgtypes.SystemContext{}

	ctx := context.Background()

	cache := blobinfocache.DefaultCache(sys)

	rawSource, err := ref.NewImageSource(ctx, sys)
	if err != nil {
		return nil, err
	}

	src, err := image.FromSource(ctx, sys, rawSource)
	if err != nil {
		if closeErr := rawSource.Close(); closeErr != nil {
			return nil, errors.Wrapf(err, " (close error: %v)", closeErr)
		}

		return nil, err
	}
	defer func() {
		if err := src.Close(); err != nil {
			retErr = errors.Wrapf(retErr, " (close error: %v)", err)
		}
	}()

	layerInfos := src.LayerInfos()

	log.Printf("Image %s has %d layers", imageName, len(layerInfos))

	// Unpack and inspect each image layer, copy relevant files to new Lambda layer
	if err := os.MkdirAll(layerOutputDir, 0777); err != nil {
		return nil, err
	}

	lambdaLayerNum := 1

	for _, layerInfo := range layerInfos {
		lambdaLayerFilename := filepath.Join(layerOutputDir, fmt.Sprintf("layer-%d.zip", lambdaLayerNum))

		layerStream, _, err := rawSource.GetBlob(ctx, layerInfo, cache)
		if err != nil {
			return nil, err
		}
		defer layerStream.Close()

		fileCreated, err := repackLayer(lambdaLayerFilename, layerStream)
		if err != nil {
			return nil, err
		}

		if fileCreated {
			log.Printf("Created Lambda layer file %s from image layer %s", lambdaLayerFilename, string(layerInfo.Digest))
			lambdaLayerNum++
			layers = append(layers, types.LambdaLayer{Digest: string(layerInfo.Digest), File: lambdaLayerFilename})
		} else {
			log.Printf("Did not create a Lambda layer file from image layer %s (no relevant files found)", string(layerInfo.Digest))
		}
	}

	log.Printf("Created %d Lambda layer files for image %s", len(layers), imageName)

	return layers, nil
}
