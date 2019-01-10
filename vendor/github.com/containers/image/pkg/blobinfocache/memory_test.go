package blobinfocache

import (
	"testing"

	"github.com/containers/image/types"
)

func newTestMemoryCache(t *testing.T) (types.BlobInfoCache, func(t *testing.T)) {
	return NewMemoryCache(), func(t *testing.T) {}
}

func TestNewMemoryCache(t *testing.T) {
	testGenericCache(t, newTestMemoryCache)
}
