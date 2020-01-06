// Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0
package extract

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/awslabs/aws-lambda-container-image-converter/img2lambda/types"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/pkg/blobinfocache"
	"github.com/containers/image/v5/transports/alltransports"
	imgtypes "github.com/containers/image/v5/types"
	zglob "github.com/mattn/go-zglob"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"
)

// Converts container image to Lambda layer and function deployment package archive files
func RepackImage(imageName string, layerOutputDir string) (layers []types.LambdaLayer, function *types.LambdaDeploymentPackage, retErr error) {
	log.Printf("Parsing the image %s", imageName)

	// Get image's layer data from image name
	ref, err := alltransports.ParseImageName(imageName)
	if err != nil {
		return nil, nil, err
	}

	sys := &imgtypes.SystemContext{}

	dockerHost := os.Getenv("DOCKER_HOST")

	// Support communicating with Docker for Windows over local plain-text TCP socket
	if dockerHost == "tcp://localhost:2375" || dockerHost == "tcp://127.0.0.1:2375" {
		sys.DockerDaemonHost = strings.Replace(dockerHost, "tcp://", "http://", -1)
	}

	// Support communicating with Docker Toolbox over encrypted socket
	if strings.HasPrefix(dockerHost, "tcp://192.168.") && strings.HasSuffix(dockerHost, ":2376") {
		sys.DockerDaemonHost = strings.Replace(dockerHost, "tcp://", "https://", -1)
	}

	ctx := context.Background()

	cache := blobinfocache.DefaultCache(sys)

	rawSource, err := ref.NewImageSource(ctx, sys)
	if err != nil {
		return nil, nil, err
	}

	src, err := image.FromSource(ctx, sys, rawSource)
	if err != nil {
		if closeErr := rawSource.Close(); closeErr != nil {
			return nil, nil, errors.Wrapf(err, " (close error: %v)", closeErr)
		}

		return nil, nil, err
	}
	defer func() {
		if err := src.Close(); err != nil {
			retErr = errors.Wrapf(retErr, " (close error: %v)", err)
		}
	}()

	return repackImage(&repackOptions{
		ctx:            ctx,
		cache:          cache,
		imageSource:    src,
		rawImageSource: rawSource,
		imageName:      imageName,
		layerOutputDir: layerOutputDir,
	})
}

type repackOptions struct {
	ctx            context.Context
	cache          imgtypes.BlobInfoCache
	imageSource    imgtypes.ImageCloser
	rawImageSource imgtypes.ImageSource
	imageName      string
	layerOutputDir string
}

func repackImage(opts *repackOptions) (layers []types.LambdaLayer, function *types.LambdaDeploymentPackage, retErr error) {

	layerInfos := opts.imageSource.LayerInfos()

	log.Printf("Image %s has %d layers", opts.imageName, len(layerInfos))

	// Unpack and inspect each image layer, copy relevant files to new Lambda layer or to a Lambda deployment package
	if err := os.MkdirAll(opts.layerOutputDir, 0777); err != nil {
		return nil, nil, err
	}

	function = &types.LambdaDeploymentPackage{FileCount: 0, File: filepath.Join(opts.layerOutputDir, "function.zip")}
	functionZip, functionFile, err := startZipFile(function.File)
	if err != nil {
		return nil, nil, fmt.Errorf("starting zip file: %v", err)
	}
	defer func() {
		if err := functionZip.Close(); err != nil {
			retErr = errors.Wrapf(err, " (zip close error: %v)", err)
		}
		if err := functionFile.Close(); err != nil {
			retErr = errors.Wrapf(err, " (file close error: %v)", err)
		}
	}()

	lambdaLayerNum := 1

	for _, layerInfo := range layerInfos {
		lambdaLayerFilename := filepath.Join(opts.layerOutputDir, fmt.Sprintf("layer-%d.zip", lambdaLayerNum))

		layerStream, _, err := opts.rawImageSource.GetBlob(opts.ctx, layerInfo, opts.cache)
		if err != nil {
			return nil, function, err
		}
		defer layerStream.Close()

		layerFileCreated, layerFunctionFileCount, err := repackLayer(lambdaLayerFilename, functionZip, layerStream, false)
		if err != nil {
			tarErr := err

			// tar extraction failed, try tar.gz
			layerStream, _, err = opts.rawImageSource.GetBlob(opts.ctx, layerInfo, opts.cache)
			if err != nil {
				return nil, function, err
			}
			defer layerStream.Close()

			layerFileCreated, layerFunctionFileCount, err = repackLayer(lambdaLayerFilename, functionZip, layerStream, true)
			if err != nil {
				return nil, function, fmt.Errorf("could not read layer with tar nor tar.gz: %v, %v", err, tarErr)
			}
		}

		function.FileCount += layerFunctionFileCount

		if layerFunctionFileCount == 0 {
			log.Printf("Did not extract any Lambda function files from image layer %s (no relevant files found)", string(layerInfo.Digest))
		}

		if layerFileCreated {
			log.Printf("Created Lambda layer file %s from image layer %s", lambdaLayerFilename, string(layerInfo.Digest))
			lambdaLayerNum++
			layers = append(layers, types.LambdaLayer{Digest: string(layerInfo.Digest), File: lambdaLayerFilename})
		} else {
			log.Printf("Did not create a Lambda layer file from image layer %s (no relevant files found)", string(layerInfo.Digest))
		}
	}

	log.Printf("Extracted %d Lambda function files for image %s", function.FileCount, opts.imageName)
	if function.FileCount > 0 {
		log.Printf("Created Lambda function deployment package %s", function.File)
	}
	log.Printf("Created %d Lambda layer files for image %s", len(layers), opts.imageName)

	return layers, function, retErr
}

// Converts container image layer archive (tar) to Lambda layer archive (zip).
// Filters files from the source and only writes a new archive if at least
// one file in the source matches the filter (i.e. does not create empty archives).
func repackLayer(outputFilename string, functionZip *archiver.Zip, layerContents io.Reader, isGzip bool) (lambdaLayerCreated bool, functionFileCount int, retError error) {
	t := archiver.NewTar()
	contentsReader := layerContents
	var err error

	if isGzip {
		gzr, err := gzip.NewReader(layerContents)
		if err != nil {
			return false, 0, fmt.Errorf("could not create gzip reader for layer: %v", err)
		}
		defer gzr.Close()
		contentsReader = gzr
	}

	err = t.Open(contentsReader, 0)
	if err != nil {
		return false, 0, fmt.Errorf("opening layer tar: %v", err)
	}
	defer t.Close()

	// Walk the files in the tar
	var z *archiver.Zip
	var out *os.File
	defer func() {
		if z != nil {
			if err := z.Close(); err != nil {
				retError = errors.Wrapf(err, " (zip close error: %v)", err)
			}
		}
		if out != nil {
			if err := out.Close(); err != nil {
				retError = errors.Wrapf(err, " (file close error: %v)", err)
			}
		}
	}()

	for {
		// Get next file in tar
		f, err := t.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return false, 0, fmt.Errorf("opening next file in layer tar: %v", err)
		}

		// Determine if this file should be repacked into a Lambda layer
		repack, err := shouldRepackLayerFileToLambdaLayer(f)
		if err != nil {
			return false, 0, fmt.Errorf("filtering file in layer tar: %v", err)
		}
		if repack {
			if z == nil {
				z, out, err = startZipFile(outputFilename)
				if err != nil {
					return false, 0, fmt.Errorf("starting zip file: %v", err)
				}
			}

			err = repackLayerFile(f, z)
		}

		if err != nil {
			return false, 0, fmt.Errorf("walking %s in layer tar: %v", f.Name(), err)
		}

		// Determine if this file should be repacked into a Lambda function package
		repack, err = shouldRepackLayerFileToLambdaFunction(f)
		if err != nil {
			return false, 0, fmt.Errorf("filtering file in layer tar: %v", err)
		}
		if repack {
			err = repackLayerFile(f, functionZip)
			functionFileCount++
		}

		if err != nil {
			return false, 0, fmt.Errorf("walking %s in layer tar: %v", f.Name(), err)
		}
	}

	return (z != nil), functionFileCount, nil
}

func startZipFile(destination string) (zip *archiver.Zip, zipFile *os.File, err error) {
	z := archiver.NewZip()

	out, err := os.Create(destination)
	if err != nil {
		return nil, nil, fmt.Errorf("creating %s: %v", destination, err)
	}

	err = z.Create(out)
	if err != nil {
		return nil, nil, fmt.Errorf("creating zip: %v", err)
	}

	return z, out, nil
}

func getLayerFileName(f archiver.File) (name string, err error) {
	header, ok := f.Header.(*tar.Header)
	if !ok {
		return "", fmt.Errorf("expected header to be *tar.Header but was %T", f.Header)
	}

	if f.IsDir() || header.Typeflag == tar.TypeDir {
		return "", nil
	}

	// Ignore whiteout files
	if strings.HasPrefix(f.Name(), ".wh.") {
		return "", nil
	}

	return header.Name, nil
}

func shouldRepackLayerFileToLambdaLayer(f archiver.File) (should bool, err error) {
	filename, err := getLayerFileName(f)
	if err != nil {
		return false, err
	}
	if filename == "" {
		return false, nil
	}

	// Only extract files that can be used for Lambda custom runtimes
	return zglob.Match("opt/**/**", filename)
}

func shouldRepackLayerFileToLambdaFunction(f archiver.File) (should bool, err error) {
	filename, err := getLayerFileName(f)
	if err != nil {
		return false, err
	}
	if filename == "" {
		return false, nil
	}

	// Only extract files that can be used for Lambda deployment packages
	return zglob.Match("var/task/**/**", filename)
}

func repackLayerFile(f archiver.File, z *archiver.Zip) error {
	hdr, ok := f.Header.(*tar.Header)
	if !ok {
		return fmt.Errorf("expected header to be *tar.Header but was %T", f.Header)
	}

	filename := strings.TrimPrefix(filepath.ToSlash(hdr.Name), "opt/")
	filename = strings.TrimPrefix(filename, "var/task/")

	switch hdr.Typeflag {
	case tar.TypeReg, tar.TypeRegA, tar.TypeChar, tar.TypeBlock, tar.TypeFifo, tar.TypeSymlink, tar.TypeLink:
		return z.Write(archiver.File{
			FileInfo: archiver.FileInfo{
				FileInfo:   f.FileInfo,
				CustomName: filename,
			},
			ReadCloser: f,
		})
	case tar.TypeXGlobalHeader:
		return nil // ignore
	default:
		return fmt.Errorf("%s: unknown type flag: %c", hdr.Name, hdr.Typeflag)
	}
}
