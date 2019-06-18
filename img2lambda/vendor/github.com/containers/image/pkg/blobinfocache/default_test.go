package blobinfocache

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/containers/image/pkg/blobinfocache/boltdb"
	"github.com/containers/image/pkg/blobinfocache/memory"
	"github.com/containers/image/types"
	"github.com/stretchr/testify/assert"
)

func TestBlobInfoCacheDir(t *testing.T) {
	const nondefaultDir = "/this/is/not/the/default/cache/dir"
	const rootPrefix = "/root/prefix"
	const homeDir = "/fake/home/directory"
	const xdgDataHome = "/fake/home/directory/XDG"

	// Environment is per-process, so this looks very unsafe; actually it seems fine because tests are not
	// run in parallel unless they opt in by calling t.Parallel().  So don’t do that.
	oldXRD, hasXRD := os.LookupEnv("XDG_RUNTIME_DIR")
	defer func() {
		if hasXRD {
			os.Setenv("XDG_RUNTIME_DIR", oldXRD)
		} else {
			os.Unsetenv("XDG_RUNTIME_DIR")
		}
	}()
	// FIXME: This should be a shared helper in internal/testing
	oldHome, hasHome := os.LookupEnv("HOME")
	defer func() {
		if hasHome {
			os.Setenv("HOME", oldHome)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	os.Setenv("HOME", homeDir)
	os.Setenv("XDG_DATA_HOME", xdgDataHome)

	// The default paths and explicit overrides
	for _, c := range []struct {
		sys      *types.SystemContext
		euid     int
		expected string
	}{
		// The common case
		{nil, 0, systemBlobInfoCacheDir},
		{nil, 1, filepath.Join(xdgDataHome, "containers", "cache")},
		// There is a context, but it does not override the path.
		{&types.SystemContext{}, 0, systemBlobInfoCacheDir},
		{&types.SystemContext{}, 1, filepath.Join(xdgDataHome, "containers", "cache")},
		// Path overridden
		{&types.SystemContext{BlobInfoCacheDir: nondefaultDir}, 0, nondefaultDir},
		{&types.SystemContext{BlobInfoCacheDir: nondefaultDir}, 1, nondefaultDir},
		// Root overridden
		{&types.SystemContext{RootForImplicitAbsolutePaths: rootPrefix}, 0, filepath.Join(rootPrefix, systemBlobInfoCacheDir)},
		{&types.SystemContext{RootForImplicitAbsolutePaths: rootPrefix}, 1, filepath.Join(xdgDataHome, "containers", "cache")},
		// Root and path overrides present simultaneously,
		{
			&types.SystemContext{
				RootForImplicitAbsolutePaths: rootPrefix,
				BlobInfoCacheDir:             nondefaultDir,
			},
			0, nondefaultDir,
		},
		{
			&types.SystemContext{
				RootForImplicitAbsolutePaths: rootPrefix,
				BlobInfoCacheDir:             nondefaultDir,
			},
			1, nondefaultDir,
		},
	} {
		path, err := blobInfoCacheDir(c.sys, c.euid)
		require.NoError(t, err)
		assert.Equal(t, c.expected, path)
	}

	// Paths used by unprivileged users
	for _, c := range []struct {
		xdgDH, home, expected string
	}{
		{"", homeDir, filepath.Join(homeDir, ".local", "share", "containers", "cache")}, // HOME only
		{xdgDataHome, "", filepath.Join(xdgDataHome, "containers", "cache")},            // XDG_DATA_HOME only
		{xdgDataHome, homeDir, filepath.Join(xdgDataHome, "containers", "cache")},       // both
		{"", "", ""}, // neither
	} {
		if c.xdgDH != "" {
			os.Setenv("XDG_DATA_HOME", c.xdgDH)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
		if c.home != "" {
			os.Setenv("HOME", c.home)
		} else {
			os.Unsetenv("HOME")
		}
		for _, sys := range []*types.SystemContext{nil, {}} {
			path, err := blobInfoCacheDir(sys, 1)
			if c.expected != "" {
				require.NoError(t, err)
				assert.Equal(t, c.expected, path)
			} else {
				assert.Error(t, err)
			}
		}
	}
}

func TestDefaultCache(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "TestDefaultCache")
	require.NoError(t, err)
	//defer os.RemoveAll(tmpDir)

	// Success
	normalDir := filepath.Join(tmpDir, "normal")
	c := DefaultCache(&types.SystemContext{BlobInfoCacheDir: normalDir})
	// This is ugly hard-coding internals of boltDBCache:
	assert.Equal(t, boltdb.New(filepath.Join(normalDir, blobInfoCacheFilename)), c)

	// Error running blobInfoCacheDir:
	// Environment is per-process, so this looks very unsafe; actually it seems fine because tests are not
	// run in parallel unless they opt in by calling t.Parallel().  So don’t do that.
	oldXRD, hasXRD := os.LookupEnv("XDG_RUNTIME_DIR")
	defer func() {
		if hasXRD {
			os.Setenv("XDG_RUNTIME_DIR", oldXRD)
		} else {
			os.Unsetenv("XDG_RUNTIME_DIR")
		}
	}()
	// FIXME: This should be a shared helper in internal/testing
	oldHome, hasHome := os.LookupEnv("HOME")
	defer func() {
		if hasHome {
			os.Setenv("HOME", oldHome)
		} else {
			os.Unsetenv("HOME")
		}
	}()
	os.Unsetenv("HOME")
	os.Unsetenv("XDG_DATA_HOME")
	c = DefaultCache(nil)
	assert.IsType(t, memory.New(), c)

	// Error creating the parent directory:
	unwritableDir := filepath.Join(tmpDir, "unwritable")
	err = os.Mkdir(unwritableDir, 700)
	require.NoError(t, err)
	defer os.Chmod(unwritableDir, 0700) // To make it possible to remove it again
	err = os.Chmod(unwritableDir, 0500)
	require.NoError(t, err)
	st, _ := os.Stat(unwritableDir)
	logrus.Errorf("%s: %#v", unwritableDir, st)
	c = DefaultCache(&types.SystemContext{BlobInfoCacheDir: filepath.Join(unwritableDir, "subdirectory")})
	assert.IsType(t, memory.New(), c)
}
