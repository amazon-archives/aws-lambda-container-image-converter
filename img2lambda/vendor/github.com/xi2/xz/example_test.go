/*
 * Package xz examples
 *
 * Author: Michael Cross <https://github.com/xi2>
 *
 * This file has been put into the public domain.
 * You can do whatever you want with this file.
 */

package xz_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/xi2/xz"
)

func ExampleNewReader() {
	// load some XZ data into memory
	data, err := ioutil.ReadFile(
		filepath.Join("testdata", "xz-utils", "good-1-check-sha256.xz"))
	if err != nil {
		log.Fatal(err)
	}
	// create an xz.Reader to decompress the data
	r, err := xz.NewReader(bytes.NewReader(data), 0)
	if err != nil {
		log.Fatal(err)
	}
	// write the decompressed data to os.Stdout
	_, err = io.Copy(os.Stdout, r)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// Hello
	// World!
}

func ExampleReader_Multistream() {
	// load some XZ data into memory
	data, err := ioutil.ReadFile(
		filepath.Join("testdata", "xz-utils", "good-1-check-sha256.xz"))
	if err != nil {
		log.Fatal(err)
	}
	// create a MultiReader that will read the data twice
	mr := io.MultiReader(bytes.NewReader(data), bytes.NewReader(data))
	// create an xz.Reader from the MultiReader
	r, err := xz.NewReader(mr, 0)
	if err != nil {
		log.Fatal(err)
	}
	// set Multistream mode to false
	r.Multistream(false)
	// decompress the first stream
	_, err = io.Copy(os.Stdout, r)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Read first stream")
	// reset the XZ reader so it is ready to read the second stream
	err = r.Reset(nil)
	if err != nil {
		log.Fatal(err)
	}
	// set Multistream mode to false again
	r.Multistream(false)
	// decompress the second stream
	_, err = io.Copy(os.Stdout, r)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Read second stream")
	// reset the XZ reader so it is ready to read further streams
	err = r.Reset(nil)
	// confirm that the second stream was the last one
	if err == io.EOF {
		fmt.Println("No more streams")
	}
	// Output:
	// Hello
	// World!
	// Read first stream
	// Hello
	// World!
	// Read second stream
	// No more streams
}
