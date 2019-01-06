package main

import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"strings"

	zglob "github.com/mattn/go-zglob"
	"github.com/mholt/archiver"
)

// Converts container image layer archive (tar) to Lambda layer archive (zip).
// Filters files from the source and only writes a new archive if at least
// one file in the source matches the filter (i.e. does not create empty archives).
func RepackLayer(outputFilename string, layerContents io.Reader, filterPattern string) (created bool, err error) {
	// TODO: support image types other than local Docker images (docker-daemon transport),
	// where the layer format is tar. For example, layers directly from a Docker registry
	// will be .tar.gz-formatted. OCI images can be either tar or tar.gz, based on the
	// layer's media type.
	t := archiver.NewTar()

	err = t.Open(layerContents, 0)
	if err != nil {
		return false, fmt.Errorf("opening layer tar: %v", err)
	}
	defer t.Close()

	// Walk the files in the tar
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
		repack, err := shouldRepackLayerFile(f, filterPattern)
		if err != nil {
			return false, fmt.Errorf("filtering file in layer tar: %v", err)
		}
		if repack {
			err = repackLayerFile(f)
		}

		if err != nil {
			return false, fmt.Errorf("walking %s in layer tar: %v", f.Name(), err)
		}
	}

	return false, nil
}

func shouldRepackLayerFile(f archiver.File, matchPattern string) (should bool, err error) {
	header, ok := f.Header.(*tar.Header)
	if !ok {
		return false, fmt.Errorf("expected header to be *tar.Header but was %T", f.Header)
	}

	if f.IsDir() {
		return false, nil
	}

	if strings.HasPrefix(f.Name(), ".wh.") {
		return false, nil
	}

	return zglob.Match(matchPattern, header.Name)
}

func repackLayerFile(f archiver.File) error {
	log.Printf(f.Header.(*tar.Header).Name)

	return nil
}
