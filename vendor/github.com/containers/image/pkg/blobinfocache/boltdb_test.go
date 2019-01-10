package blobinfocache

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/containers/image/types"
	"github.com/stretchr/testify/require"
)

func newTestBoltDBCache(t *testing.T) (types.BlobInfoCache, func(t *testing.T)) {
	// We need a separate temporary directory here, because bolt.Open(…, &bolt.Options{Readonly:true}) can't deal with
	// an existing but empty file, and incorrectly fails without releasing the lock - which in turn causes
	// any future writes to hang.  Creating a temporary directory allows us to use a path to a
	// non-existent file, thus replicating the expected conditions for creating a new DB.
	dir, err := ioutil.TempDir("", "boltdb")
	require.NoError(t, err)
	return NewBoltDBCache(filepath.Join(dir, "db")), func(t *testing.T) {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}
}

func TestNewBoltDBCache(t *testing.T) {
	testGenericCache(t, newTestBoltDBCache)
}

// FIXME: Tests for the various corner cases / failure cases of boltDBCache should be added here.
