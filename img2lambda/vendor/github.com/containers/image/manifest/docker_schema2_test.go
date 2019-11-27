package manifest

import (
	"io/ioutil"
	"testing"

	"github.com/containers/image/pkg/compression"
	"github.com/containers/image/types"
	"github.com/stretchr/testify/assert"
)

func TestSupportedSchema2MediaType(t *testing.T) {
	type testData struct {
		m        string
		mustFail bool
	}
	data := []testData{
		{
			DockerV2Schema2MediaType,
			false,
		},
		{
			DockerV2Schema2ConfigMediaType,
			false,
		},
		{
			DockerV2Schema2LayerMediaType,
			false,
		},
		{
			DockerV2SchemaLayerMediaTypeUncompressed,
			false,
		},
		{
			DockerV2ListMediaType,
			false,
		},
		{
			DockerV2Schema2ForeignLayerMediaType,
			false,
		},
		{
			DockerV2Schema2ForeignLayerMediaTypeGzip,
			false,
		},
		{
			"application/vnd.docker.image.rootfs.foreign.diff.unknown",
			true,
		},
	}
	for _, d := range data {
		err := SupportedSchema2MediaType(d.m)
		if d.mustFail {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestUpdateLayerInfosV2S2GzipToZstd(t *testing.T) {
	bytes, err := ioutil.ReadFile("fixtures/v2s2.manifest.json")
	assert.Nil(t, err)

	origManifest, err := Schema2FromManifest(bytes)
	assert.Nil(t, err)

	err = origManifest.UpdateLayerInfos([]types.BlobInfo{
		{
			Digest:               "sha256:e692418e4cbaf90ca69d05a66403747baa33ee08806650b51fab815ad7fc331f",
			Size:                 32654,
			MediaType:            DockerV2Schema2LayerMediaType,
			CompressionOperation: types.Compress,
			CompressionAlgorithm: &compression.Zstd,
		},
		{
			Digest:               "sha256:3c3a4604a545cdc127456d94e421cd355bca5b528f4a9c1905b15da2eb4a4c6b",
			Size:                 16724,
			MediaType:            DockerV2Schema2LayerMediaType,
			CompressionOperation: types.Compress,
			CompressionAlgorithm: &compression.Zstd,
		},
		{
			Digest:               "sha256:ec4b8955958665577945c89419d1af06b5f7636b4ac3da7f12184802ad867736",
			Size:                 73109,
			MediaType:            DockerV2Schema2LayerMediaType,
			CompressionOperation: types.Compress,
			CompressionAlgorithm: &compression.Zstd,
		},
	})
	assert.NotNil(t, err) // zstd is not supported for docker images
}

func TestUpdateLayerInfosV2S2InvalidCompressionOperation(t *testing.T) {
	bytes, err := ioutil.ReadFile("fixtures/v2s2.manifest.json")
	assert.Nil(t, err)

	origManifest, err := Schema2FromManifest(bytes)
	assert.Nil(t, err)

	err = origManifest.UpdateLayerInfos([]types.BlobInfo{
		{
			Digest:               "sha256:e692418e4cbaf90ca69d05a66403747baa33ee08806650b51fab815ad7fc331f",
			Size:                 32654,
			MediaType:            DockerV2Schema2LayerMediaType,
			CompressionOperation: types.Decompress,
		},
		{
			Digest:               "sha256:3c3a4604a545cdc127456d94e421cd355bca5b528f4a9c1905b15da2eb4a4c6b",
			Size:                 16724,
			MediaType:            DockerV2Schema2LayerMediaType,
			CompressionOperation: types.Decompress,
		},
		{
			Digest:               "sha256:ec4b8955958665577945c89419d1af06b5f7636b4ac3da7f12184802ad867736",
			Size:                 73109,
			MediaType:            DockerV2Schema2LayerMediaType,
			CompressionOperation: 42, // MUST fail here
		},
	})
	assert.NotNil(t, err)
}

func TestUpdateLayerInfosV2S2InvalidCompressionAlgorithm(t *testing.T) {
	bytes, err := ioutil.ReadFile("fixtures/v2s2.manifest.json")
	assert.Nil(t, err)

	origManifest, err := Schema2FromManifest(bytes)
	assert.Nil(t, err)

	err = origManifest.UpdateLayerInfos([]types.BlobInfo{
		{
			Digest:               "sha256:e692418e4cbaf90ca69d05a66403747baa33ee08806650b51fab815ad7fc331f",
			Size:                 32654,
			MediaType:            DockerV2Schema2LayerMediaType,
			CompressionOperation: types.Compress,
			CompressionAlgorithm: &compression.Gzip,
		},
		{
			Digest:               "sha256:3c3a4604a545cdc127456d94e421cd355bca5b528f4a9c1905b15da2eb4a4c6b",
			Size:                 16724,
			MediaType:            DockerV2Schema2LayerMediaType,
			CompressionOperation: types.Compress,
			CompressionAlgorithm: &compression.Gzip,
		},
		{
			Digest:               "sha256:ec4b8955958665577945c89419d1af06b5f7636b4ac3da7f12184802ad867736",
			Size:                 73109,
			MediaType:            DockerV2Schema2LayerMediaType,
			CompressionOperation: types.Compress,
			CompressionAlgorithm: &compression.Zstd, // MUST fail here
		},
	})
	assert.NotNil(t, err)
}

func TestUpdateLayerInfosV2S2NondistributableToGzip(t *testing.T) {
	bytes, err := ioutil.ReadFile("fixtures/v2s2.nondistributable.manifest.json")
	assert.Nil(t, err)

	origManifest, err := Schema2FromManifest(bytes)
	assert.Nil(t, err)

	err = origManifest.UpdateLayerInfos([]types.BlobInfo{
		{
			Digest:               "sha256:e692418e4cbaf90ca69d05a66403747baa33ee08806650b51fab815ad7fc331f",
			Size:                 32654,
			MediaType:            DockerV2Schema2ForeignLayerMediaType,
			CompressionOperation: types.Compress,
			CompressionAlgorithm: &compression.Gzip,
		},
	})
	assert.Nil(t, err)

	updatedManifestBytes, err := origManifest.Serialize()
	assert.Nil(t, err)

	bytes, err = ioutil.ReadFile("fixtures/v2s2.nondistributable.gzip.manifest.json")
	assert.Nil(t, err)

	expectedManifest, err := Schema2FromManifest(bytes)
	assert.Nil(t, err)

	expectedManifestBytes, err := expectedManifest.Serialize()
	assert.Nil(t, err)

	assert.Equal(t, string(expectedManifestBytes), string(updatedManifestBytes))
}

func TestUpdateLayerInfosV2S2NondistributableGzipToUncompressed(t *testing.T) {
	bytes, err := ioutil.ReadFile("fixtures/v2s2.nondistributable.gzip.manifest.json")
	assert.Nil(t, err)

	origManifest, err := Schema2FromManifest(bytes)
	assert.Nil(t, err)

	err = origManifest.UpdateLayerInfos([]types.BlobInfo{
		{
			Digest:               "sha256:e692418e4cbaf90ca69d05a66403747baa33ee08806650b51fab815ad7fc331f",
			Size:                 32654,
			MediaType:            DockerV2Schema2ForeignLayerMediaType,
			CompressionOperation: types.Decompress,
		},
	})
	assert.Nil(t, err)

	updatedManifestBytes, err := origManifest.Serialize()
	assert.Nil(t, err)

	bytes, err = ioutil.ReadFile("fixtures/v2s2.nondistributable.manifest.json")
	assert.Nil(t, err)

	expectedManifest, err := Schema2FromManifest(bytes)
	assert.Nil(t, err)

	expectedManifestBytes, err := expectedManifest.Serialize()
	assert.Nil(t, err)

	assert.Equal(t, string(expectedManifestBytes), string(updatedManifestBytes))
}
