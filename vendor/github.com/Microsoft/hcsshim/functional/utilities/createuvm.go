package testutilities

import (
	"os"
	"testing"

	"github.com/Microsoft/hcsshim/internal/guid"
	"github.com/Microsoft/hcsshim/internal/uvm"
)

// CreateWCOWUVM creates a WCOW utility VM with all default options. Returns the
// UtilityVM object; folder used as its scratch
func CreateWCOWUVM(t *testing.T, id, image string) (*uvm.UtilityVM, []string, string) {
	return CreateWCOWUVMFromOptsWithImage(t, &uvm.UVMOptions{ID: id, OperatingSystem: "windows"}, image)
}

// CreateWCOWUVMFromOpts creates a WCOW utility VM with the passed opts.
func CreateWCOWUVMFromOpts(t *testing.T, opts *uvm.UVMOptions) *uvm.UtilityVM {
	if opts == nil || len(opts.LayerFolders) < 2 {
		t.Fatalf("opts must bet set with LayerFolders")
	}
	if opts.ID == "" {
		opts.ID = guid.New().String()
	}

	uvm, err := uvm.Create(opts)
	if err != nil {
		t.Fatal(err)
	}
	if err := uvm.Start(); err != nil {
		uvm.Close()
		t.Fatal(err)
	}
	return uvm
}

// CreateWCOWUVMFromOptsWithImage creates a WCOW utility VM with the passed opts
// builds the LayerFolders based on `image`. Returns the UtilityVM object;
// folder used as its scratch
func CreateWCOWUVMFromOptsWithImage(t *testing.T, opts *uvm.UVMOptions, image string) (*uvm.UtilityVM, []string, string) {
	if opts == nil {
		t.Fatal("opts must be set")
	}

	uvmLayers := LayerFolders(t, image)
	scratchDir := CreateTempDir(t)
	defer func() {
		if t.Failed() {
			os.RemoveAll(scratchDir)
		}
	}()

	opts.LayerFolders = append(opts.LayerFolders, uvmLayers...)
	opts.LayerFolders = append(opts.LayerFolders, scratchDir)

	return CreateWCOWUVMFromOpts(t, opts), uvmLayers, scratchDir
}

// CreateLCOWUVM with all default options.
func CreateLCOWUVM(t *testing.T, id string) *uvm.UtilityVM {
	return CreateLCOWUVMFromOpts(t, &uvm.UVMOptions{ID: id, OperatingSystem: "linux"})
}

// CreateLCOWUVMFromOpts creates an LCOW utility VM with the specified options.
func CreateLCOWUVMFromOpts(t *testing.T, opts *uvm.UVMOptions) *uvm.UtilityVM {
	if opts == nil {
		opts = &uvm.UVMOptions{}
	}
	if opts.ID == "" {
		opts.ID = guid.New().String()
	}

	uvm, err := uvm.Create(opts)
	if err != nil {
		t.Fatal(err)
	}
	if err := uvm.Start(); err != nil {
		uvm.Close()
		t.Fatal(err)
	}
	return uvm
}
