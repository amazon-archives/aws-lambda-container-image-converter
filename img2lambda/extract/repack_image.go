package extract

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/awslabs/aws-lambda-container-image-converter/img2lambda/types"
	"github.com/containers/image/image"
	"github.com/containers/image/pkg/blobinfocache"
	"github.com/containers/image/transports/alltransports"
	imgtypes "github.com/containers/image/types"
	zglob "github.com/mattn/go-zglob"
	"github.com/mholt/archiver"
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

func repackImage(opts *repackOptions) (layers []types.LambdaLayer, retErr error) {

	layerInfos := opts.imageSource.LayerInfos()

	log.Printf("Image %s has %d layers", opts.imageName, len(layerInfos))

	// Unpack and inspect each image layer, copy relevant files to new Lambda layer
	if err := os.MkdirAll(opts.layerOutputDir, 0777); err != nil {
		return nil, err
	}

	lambdaLayerNum := 1

	for _, layerInfo := range layerInfos {
		lambdaLayerFilename := filepath.Join(opts.layerOutputDir, fmt.Sprintf("layer-%d.zip", lambdaLayerNum))

		layerStream, _, err := opts.rawImageSource.GetBlob(opts.ctx, layerInfo, opts.cache)
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

	log.Printf("Created %d Lambda layer files for image %s", len(layers), opts.imageName)

	return layers, nil
}

// Converts container image layer archive (tar) to Lambda layer archive (zip).
// Filters files from the source and only writes a new archive if at least
// one file in the source matches the filter (i.e. does not create empty archives).
func repackLayer(outputFilename string, layerContents io.Reader) (created bool, retError error) {
	t := archiver.NewTar()

	err := t.Open(layerContents, 0)
	if err != nil {
		return false, fmt.Errorf("opening layer tar: %v", err)
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
			return false, fmt.Errorf("opening next file in layer tar: %v", err)
		}

		// Determine if this file should be repacked
		repack, err := shouldRepackLayerFile(f)
		if err != nil {
			return false, fmt.Errorf("filtering file in layer tar: %v", err)
		}
		if repack {
			if z == nil {
				z, out, err = startZipFile(outputFilename)
				if err != nil {
					return false, fmt.Errorf("starting zip file: %v", err)
				}
			}

			err = repackLayerFile(f, z)
		}

		if err != nil {
			return false, fmt.Errorf("walking %s in layer tar: %v", f.Name(), err)
		}
	}

	return (z != nil), nil
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

func shouldRepackLayerFile(f archiver.File) (should bool, err error) {
	header, ok := f.Header.(*tar.Header)
	if !ok {
		return false, fmt.Errorf("expected header to be *tar.Header but was %T", f.Header)
	}

	if f.IsDir() || header.Typeflag == tar.TypeDir {
		return false, nil
	}

	// Ignore whiteout files
	if strings.HasPrefix(f.Name(), ".wh.") {
		return false, nil
	}

	// Only extract files that can be used for Lambda custom runtimes
	return zglob.Match("opt/**/**", header.Name)
}

func repackLayerFile(f archiver.File, z *archiver.Zip) error {
	hdr, ok := f.Header.(*tar.Header)
	if !ok {
		return fmt.Errorf("expected header to be *tar.Header but was %T", f.Header)
	}

	filename := strings.TrimPrefix(filepath.ToSlash(hdr.Name), "opt/")

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
