// +build functional uvmmem

package functional

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/Microsoft/hcsshim/functional/utilities"
	"github.com/Microsoft/hcsshim/internal/uvm"
	"github.com/Microsoft/hcsshim/osversion"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
)

func runMemStartLCOWTest(t *testing.T, opts *uvm.UVMOptions) {
	u := testutilities.CreateLCOWUVMFromOpts(t, opts)
	u.Close()
}

func runMemStartWCOWTest(t *testing.T, opts *uvm.UVMOptions) {
	u, _, scratchDir := testutilities.CreateWCOWUVMFromOptsWithImage(t, opts, "microsoft/nanoserver")
	defer os.RemoveAll(scratchDir)
	u.Close()
}

func runMemTests(t *testing.T, os string) {
	type testCase struct {
		allowOvercommit      *bool
		enableDeferredCommit *bool
	}

	yes := true
	no := false

	testCases := []testCase{
		{nil, nil}, // Implicit default - Virtual
		{allowOvercommit: &yes, enableDeferredCommit: &no},  // Explicit default - Virtual
		{allowOvercommit: &yes, enableDeferredCommit: &yes}, // Virtual Deferred
		{allowOvercommit: &no, enableDeferredCommit: &no},   // Physical
	}

	mem := uint64(512 * 1024 * 1024) // 512 MB (OCI in Bytes)
	for _, bt := range testCases {
		opts := &uvm.UVMOptions{
			ID:              t.Name(),
			OperatingSystem: os,
			Resources: &specs.WindowsResources{
				Memory: &specs.WindowsMemoryResources{
					Limit: &mem,
				},
			},
			AllowOvercommit:      bt.allowOvercommit,
			EnableDeferredCommit: bt.enableDeferredCommit,
		}

		if os == "windows" {
			runMemStartWCOWTest(t, opts)
		} else {
			runMemStartLCOWTest(t, opts)
		}
	}
}

func TestMemBackingTypeWCOW(t *testing.T) {
	testutilities.RequiresBuild(t, osversion.RS5)
	runMemTests(t, "windows")
}

func TestMemBackingTypeLCOW(t *testing.T) {
	testutilities.RequiresBuild(t, osversion.RS5)
	runMemTests(t, "linux")
}

func runBenchMemStartTest(b *testing.B, opts *uvm.UVMOptions) {
	u, err := uvm.Create(opts)
	if err != nil {
		b.Fatal(err)
	}
	defer u.Close()
	if err := u.Start(); err != nil {
		b.Fatal(err)
	}
}

func runBenchMemStartLcowTest(b *testing.B, allowOverCommit bool, enableDeferredCommit bool) {
	mem := uint64(512 * 1024 * 1024) // 512 MB (OCI in Bytes)
	for i := 0; i < b.N; i++ {
		opts := &uvm.UVMOptions{
			ID:              b.Name(),
			OperatingSystem: "linux",
			Resources: &specs.WindowsResources{
				Memory: &specs.WindowsMemoryResources{
					Limit: &mem,
				},
			},
			AllowOvercommit:      &allowOverCommit,
			EnableDeferredCommit: &enableDeferredCommit,
		}
		runBenchMemStartTest(b, opts)
	}
}

func BenchmarkMemBackingTypeVirtualLCOW(b *testing.B) {
	//testutilities.RequiresBuild(t, osversion.RS5)
	logrus.SetOutput(ioutil.Discard)

	runBenchMemStartLcowTest(b, true, false)
}

func BenchmarkMemBackingTypeVirtualDeferredLCOW(b *testing.B) {
	//testutilities.RequiresBuild(t, osversion.RS5)
	logrus.SetOutput(ioutil.Discard)

	runBenchMemStartLcowTest(b, true, true)
}

func BenchmarkMemBackingTypePhyscialLCOW(b *testing.B) {
	//testutilities.RequiresBuild(t, osversion.RS5)
	logrus.SetOutput(ioutil.Discard)

	runBenchMemStartLcowTest(b, false, false)
}
