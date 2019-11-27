package sysregistriesv2

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/containers/image/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/containers/image/docker/reference"
)

func TestParseLocation(t *testing.T) {
	var err error
	var location string

	// invalid locations
	_, err = parseLocation("https://example.com")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid location 'https://example.com': URI schemes are not supported")

	_, err = parseLocation("john.doe@example.com")
	assert.Nil(t, err)

	// valid locations
	location, err = parseLocation("example.com")
	assert.Nil(t, err)
	assert.Equal(t, "example.com", location)

	location, err = parseLocation("example.com/") // trailing slashes are stripped
	assert.Nil(t, err)
	assert.Equal(t, "example.com", location)

	location, err = parseLocation("example.com//////") // trailing slahes are stripped
	assert.Nil(t, err)
	assert.Equal(t, "example.com", location)

	location, err = parseLocation("example.com:5000/with/path")
	assert.Nil(t, err)
	assert.Equal(t, "example.com:5000/with/path", location)
}

func TestEmptyConfig(t *testing.T) {
	registries, err := GetRegistries(&types.SystemContext{SystemRegistriesConfPath: "testdata/empty.conf"})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(registries))

	// When SystemRegistriesConfPath is not explicitly specified (but RootForImplicitAbsolutePaths might be), missing file is treated
	// the same as an empty one, without reporting an error.
	nonexistentRoot, err := filepath.Abs("testdata/this-does-not-exist")
	require.NoError(t, err)
	registries, err = GetRegistries(&types.SystemContext{RootForImplicitAbsolutePaths: nonexistentRoot})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(registries))
}

func TestMirrors(t *testing.T) {
	sys := &types.SystemContext{SystemRegistriesConfPath: "testdata/mirrors.conf"}

	registries, err := GetRegistries(sys)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(registries))

	reg, err := FindRegistry(sys, "registry.com/image:tag")
	assert.Nil(t, err)
	assert.NotNil(t, reg)
	assert.Equal(t, 2, len(reg.Mirrors))
	assert.Equal(t, "mirror-1.registry.com", reg.Mirrors[0].Location)
	assert.False(t, reg.Mirrors[0].Insecure)
	assert.Equal(t, "mirror-2.registry.com", reg.Mirrors[1].Location)
	assert.True(t, reg.Mirrors[1].Insecure)
}

func TestRefMatchesPrefix(t *testing.T) {
	for _, c := range []struct {
		ref, prefix string
		expected    bool
	}{
		// Prefix is a reference.Domain() value
		{"docker.io", "docker.io", true},
		{"docker.io", "example.com", false},
		{"example.com:5000", "example.com:5000", true},
		{"example.com:50000", "example.com:5000", false},
		{"example.com:5000", "example.com", true}, // FIXME FIXME This is unintended and undocumented, don't rely on this behavior
		{"example.com/foo", "example.com", true},
		{"example.com/foo/bar", "example.com", true},
		{"example.com/foo/bar:baz", "example.com", true},
		{"example.com/foo/bar@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "example.com", true},
		// Prefix is a reference.Named.Name() value or a repo namespace
		{"docker.io", "docker.io/library", false},
		{"docker.io/library", "docker.io/library", true},
		{"example.com/library", "docker.io/library", false},
		{"docker.io/libraryy", "docker.io/library", false},
		{"docker.io/library/busybox", "docker.io/library", true},
		{"docker.io", "docker.io/library/busybox", false},
		{"docker.io/library/busybox", "docker.io/library/busybox", true},
		{"example.com/library/busybox", "docker.io/library/busybox", false},
		{"docker.io/library/busybox2", "docker.io/library/busybox", false},
		// Prefix is a single image
		{"example.com", "example.com/foo:bar", false},
		{"example.com/foo", "example.com/foo:bar", false},
		{"example.com/foo:bar", "example.com/foo:bar", true},
		{"example.com/foo:bar2", "example.com/foo:bar", false},
		{"example.com", "example.com/foo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},
		{"example.com/foo", "example.com/foo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},
		{"example.com/foo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "example.com/foo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
		{"example.com/foo@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "example.com/foo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},
	} {
		res := refMatchesPrefix(c.ref, c.prefix)
		assert.Equal(t, c.expected, res, fmt.Sprintf("%s vs. %s", c.ref, c.prefix))
	}
}

func TestConfigPath(t *testing.T) {
	const nondefaultPath = "/this/is/not/the/default/registries.conf"
	const variableReference = "$HOME"
	const rootPrefix = "/root/prefix"

	for _, c := range []struct {
		sys      *types.SystemContext
		expected string
	}{
		// The common case
		{nil, systemRegistriesConfPath},
		// There is a context, but it does not override the path.
		{&types.SystemContext{}, systemRegistriesConfPath},
		// Path overridden
		{&types.SystemContext{SystemRegistriesConfPath: nondefaultPath}, nondefaultPath},
		// Root overridden
		{
			&types.SystemContext{RootForImplicitAbsolutePaths: rootPrefix},
			filepath.Join(rootPrefix, systemRegistriesConfPath),
		},
		// Root and path overrides present simultaneously,
		{
			&types.SystemContext{
				RootForImplicitAbsolutePaths: rootPrefix,
				SystemRegistriesConfPath:     nondefaultPath,
			},
			nondefaultPath,
		},
		// No environment expansion happens in the overridden paths
		{&types.SystemContext{SystemRegistriesConfPath: variableReference}, variableReference},
	} {
		path := ConfigPath(c.sys)
		assert.Equal(t, c.expected, path)
	}
}

func TestFindRegistry(t *testing.T) {
	sys := &types.SystemContext{SystemRegistriesConfPath: "testdata/find-registry.conf"}

	registries, err := GetRegistries(sys)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(registries))

	reg, err := FindRegistry(sys, "simple-prefix.com/foo/bar:latest")
	assert.Nil(t, err)
	assert.NotNil(t, reg)
	assert.Equal(t, "simple-prefix.com", reg.Prefix)
	assert.Equal(t, reg.Location, "registry.com:5000")

	// path match
	reg, err = FindRegistry(sys, "simple-prefix.com/")
	assert.Nil(t, err)
	assert.NotNil(t, reg)

	// hostname match
	reg, err = FindRegistry(sys, "simple-prefix.com")
	assert.Nil(t, err)
	assert.NotNil(t, reg)

	// invalid match
	reg, err = FindRegistry(sys, "simple-prefix.comx")
	assert.Nil(t, err)
	assert.Nil(t, reg)

	reg, err = FindRegistry(sys, "complex-prefix.com:4000/with/path/and/beyond:tag")
	assert.Nil(t, err)
	assert.NotNil(t, reg)
	assert.Equal(t, "complex-prefix.com:4000/with/path", reg.Prefix)
	assert.Equal(t, "another-registry.com:5000", reg.Location)

	reg, err = FindRegistry(sys, "no-prefix.com/foo:tag")
	assert.Nil(t, err)
	assert.NotNil(t, reg)
	assert.Equal(t, "no-prefix.com", reg.Prefix)
	assert.Equal(t, "no-prefix.com", reg.Location)

	reg, err = FindRegistry(sys, "empty-prefix.com/foo:tag")
	assert.Nil(t, err)
	assert.NotNil(t, reg)
	assert.Equal(t, "empty-prefix.com", reg.Prefix)
	assert.Equal(t, "empty-prefix.com", reg.Location)

	_, err = FindRegistry(&types.SystemContext{SystemRegistriesConfPath: "testdata/this-does-not-exist.conf"}, "example.com")
	assert.Error(t, err)
}

func assertRegistryLocationsEqual(t *testing.T, expected []string, regs []Registry) {
	// verify the expected registries and their order
	names := []string{}
	for _, r := range regs {
		names = append(names, r.Location)
	}
	assert.Equal(t, expected, names)
}

func TestFindUnqualifiedSearchRegistries(t *testing.T) {
	sys := &types.SystemContext{SystemRegistriesConfPath: "testdata/unqualified-search.conf"}

	registries, err := GetRegistries(sys)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(registries))

	unqRegs, err := UnqualifiedSearchRegistries(sys)
	assert.Nil(t, err)
	assert.Equal(t, []string{"registry-a.com", "registry-c.com", "registry-d.com"}, unqRegs)

	_, err = UnqualifiedSearchRegistries(&types.SystemContext{SystemRegistriesConfPath: "testdata/invalid-search.conf"})
	assert.Error(t, err)
}

func TestInvalidV2Configs(t *testing.T) {
	for _, c := range []struct{ path, errorSubstring string }{
		{"testdata/insecure-conflicts.conf", "registry 'registry.com' is defined multiple times with conflicting 'insecure' setting"},
		{"testdata/blocked-conflicts.conf", "registry 'registry.com' is defined multiple times with conflicting 'blocked' setting"},
		{"testdata/missing-registry-location.conf", "invalid location"},
		{"testdata/missing-mirror-location.conf", "invalid location"},
		{"testdata/invalid-prefix.conf", "invalid location"},
		{"testdata/this-does-not-exist.conf", "no such file or directory"},
	} {
		_, err := GetRegistries(&types.SystemContext{SystemRegistriesConfPath: c.path})
		assert.Error(t, err, c.path)
		if c.errorSubstring != "" {
			assert.Contains(t, err.Error(), c.errorSubstring, c.path)
		}
	}
}

func TestUnmarshalConfig(t *testing.T) {
	registries, err := GetRegistries(&types.SystemContext{SystemRegistriesConfPath: "testdata/unmarshal.conf"})
	assert.Nil(t, err)
	assert.Equal(t, 4, len(registries))
}

func TestV1BackwardsCompatibility(t *testing.T) {
	sys := &types.SystemContext{SystemRegistriesConfPath: "testdata/v1-compatibility.conf"}

	registries, err := GetRegistries(sys)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(registries))

	unqRegs, err := UnqualifiedSearchRegistries(sys)
	assert.Nil(t, err)
	assert.Equal(t, []string{"registry-a.com", "registry-c.com", "registry-d.com"}, unqRegs)

	// check if merging works
	reg, err := FindRegistry(sys, "registry-b.com/bar/foo/barfoo:latest")
	assert.Nil(t, err)
	assert.NotNil(t, reg)
	assert.True(t, reg.Insecure)
	assert.True(t, reg.Blocked)

	for _, c := range []string{"testdata/v1-invalid-block.conf", "testdata/v1-invalid-insecure.conf", "testdata/v1-invalid-search.conf"} {
		_, err := GetRegistries(&types.SystemContext{SystemRegistriesConfPath: c})
		assert.Error(t, err, c)
	}
}

func TestMixingV1andV2(t *testing.T) {
	_, err := GetRegistries(&types.SystemContext{SystemRegistriesConfPath: "testdata/mixing-v1-v2.conf"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "mixing sysregistry v1/v2 is not supported")
}

func TestConfigCache(t *testing.T) {
	configFile, err := ioutil.TempFile("", "sysregistriesv2-test")
	require.NoError(t, err)
	defer os.Remove(configFile.Name())
	defer configFile.Close()

	err = ioutil.WriteFile(configFile.Name(), []byte(`
[[registry]]
location = "registry.com"

[[registry.mirror]]
location = "mirror-1.registry.com"

[[registry.mirror]]
location = "mirror-2.registry.com"


[[registry]]
location = "blocked.registry.com"
blocked = true


[[registry]]
location = "insecure.registry.com"
insecure = true


[[registry]]
location = "untrusted.registry.com"
insecure = true`), 0600)
	require.NoError(t, err)

	ctx := &types.SystemContext{SystemRegistriesConfPath: configFile.Name()}

	configCache = make(map[string]*V2RegistriesConf)
	registries, err := GetRegistries(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(registries))

	// empty the config, but use the same SystemContext to show that the
	// previously specified registries are in the cache
	err = ioutil.WriteFile(configFile.Name(), []byte{}, 0600)
	require.NoError(t, err)
	registries, err = GetRegistries(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(registries))
}

func TestInvalidateCache(t *testing.T) {
	ctx := &types.SystemContext{SystemRegistriesConfPath: "testdata/invalidate-cache.conf"}

	configCache = make(map[string]*V2RegistriesConf)
	registries, err := GetRegistries(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(registries))
	assertRegistryLocationsEqual(t, []string{"registry.com", "blocked.registry.com", "insecure.registry.com", "untrusted.registry.com"}, registries)

	// invalidate the cache, make sure it's empty and reload
	InvalidateCache()
	assert.Equal(t, 0, len(configCache))

	registries, err = GetRegistries(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(registries))
	assertRegistryLocationsEqual(t, []string{"registry.com", "blocked.registry.com", "insecure.registry.com", "untrusted.registry.com"}, registries)
}

func toNamedRef(t *testing.T, ref string) reference.Named {
	parsedRef, err := reference.ParseNamed(ref)
	require.NoError(t, err)
	return parsedRef
}

func TestRewriteReferenceSuccess(t *testing.T) {
	for _, c := range []struct{ inputRef, prefix, location, expected string }{
		// Standard use cases
		{"example.com/image", "example.com", "example.com", "example.com/image"},
		{"example.com/image:latest", "example.com", "example.com", "example.com/image:latest"},
		{"example.com:5000/image", "example.com:5000", "example.com:5000", "example.com:5000/image"},
		{"example.com:5000/image:latest", "example.com:5000", "example.com:5000", "example.com:5000/image:latest"},
		// Separator test ('/', '@', ':')
		{"example.com/foo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"example.com", "example.com",
			"example.com/foo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{"example.com/foo/image:latest", "example.com/foo", "example.com", "example.com/image:latest"},
		{"example.com/foo/image:latest", "example.com/foo", "example.com/path", "example.com/path/image:latest"},
		// Docker examples
		{"docker.io/library/image:latest", "docker.io", "docker.io", "docker.io/library/image:latest"},
		{"docker.io/library/image", "docker.io/library", "example.com", "example.com/image"},
		{"docker.io/library/image", "docker.io", "example.com", "example.com/library/image"},
		{"docker.io/library/prefix/image", "docker.io/library/prefix", "example.com", "example.com/image"},
	} {
		ref := toNamedRef(t, c.inputRef)
		testEndpoint := Endpoint{Location: c.location}
		out, err := testEndpoint.rewriteReference(ref, c.prefix)
		require.NoError(t, err)
		assert.Equal(t, c.expected, out.String())
	}
}

func TestRewriteReferenceFailedDuringParseNamed(t *testing.T) {
	for _, c := range []struct{ inputRef, prefix, location string }{
		// Invalid reference format
		{"example.com/foo/image:latest", "example.com/foo", "example.com/path/"},
		{"example.com/foo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"example.com/foo", "example.com"},
		{"example.com:5000/image:latest", "example.com", ""},
		{"example.com:5000/image:latest", "example.com", "example.com:5000"},
		// Malformed prefix
		{"example.com/foo/image:latest", "example.com//foo", "example.com/path"},
		{"example.com/image:latest", "image", "anotherimage"},
		{"example.com/foo/image:latest", "example.com/foo/", "example.com"},
		{"example.com/foo/image", "example.com/fo", "example.com/foo"},
		{"example.com/foo:latest", "example.com/fo", "example.com/foo"},
		{"example.com/foo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"example.com/fo", "example.com/foo"},
		{"docker.io/library/image", "example.com", "example.com"},
	} {
		ref := toNamedRef(t, c.inputRef)
		testEndpoint := Endpoint{Location: c.location}
		out, err := testEndpoint.rewriteReference(ref, c.prefix)
		assert.NotNil(t, err)
		assert.Nil(t, out)
	}
}

func TestPullSourcesFromReference(t *testing.T) {
	sys := &types.SystemContext{SystemRegistriesConfPath: "testdata/pull-sources-from-reference.conf"}
	registries, err := GetRegistries(sys)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(registries))

	// Registry A allowing any kind of pull from mirrors
	registryA, err := FindRegistry(sys, "registry-a.com/foo/image:latest")
	assert.Nil(t, err)
	assert.NotNil(t, registryA)
	// Digest
	referenceADigest := toNamedRef(t, "registry-a.com/foo/image@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	pullSources, err := registryA.PullSourcesFromReference(referenceADigest)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(pullSources))
	assert.Equal(t, "mirror-1.registry-a.com/image@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", pullSources[0].Reference.String())
	assert.True(t, pullSources[1].Endpoint.Insecure)
	// Tag
	referenceATag := toNamedRef(t, "registry-a.com/foo/image:aaa")
	pullSources, err = registryA.PullSourcesFromReference(referenceATag)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(pullSources))
	assert.Equal(t, "registry-a.com/bar/image:aaa", pullSources[2].Reference.String())

	// Registry B allowing digests pull only from mirrors
	registryB, err := FindRegistry(sys, "registry-b.com/foo/image:latest")
	assert.Nil(t, err)
	assert.NotNil(t, registryB)
	// Digest
	referenceBDigest := toNamedRef(t, "registry-b.com/foo/image@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	pullSources, err = registryB.PullSourcesFromReference(referenceBDigest)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(pullSources))
	assert.Equal(t, "registry-b.com/bar", pullSources[2].Endpoint.Location)
	assert.Equal(t, "registry-b.com/bar/image@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", pullSources[2].Reference.String())
	// Tag
	referenceBTag := toNamedRef(t, "registry-b.com/foo/image:aaa")
	pullSources, err = registryB.PullSourcesFromReference(referenceBTag)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(pullSources))
}

func TestTryUpdatingCache(t *testing.T) {
	ctx := &types.SystemContext{
		SystemRegistriesConfPath: "testdata/try-update-cache-valid.conf",
	}
	configCache = make(map[string]*V2RegistriesConf)
	registries, err := TryUpdatingCache(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(registries.Registries))
	assert.Equal(t, 1, len(configCache))

	ctxInvalid := &types.SystemContext{
		SystemRegistriesConfPath: "testdata/try-update-cache-invalid.conf",
	}
	registries, err = TryUpdatingCache(ctxInvalid)
	assert.NotNil(t, err)
	assert.Nil(t, registries)
	assert.Equal(t, 1, len(configCache))
}
