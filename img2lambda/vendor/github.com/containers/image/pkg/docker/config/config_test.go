package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/containers/image/types"
	"github.com/containers/storage/pkg/homedir"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPathToAuth(t *testing.T) {
	uid := fmt.Sprintf("%d", os.Getuid())

	tmpDir, err := ioutil.TempDir("", "TestGetPathToAuth")
	require.NoError(t, err)

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

	for _, c := range []struct {
		sys      *types.SystemContext
		xrd      string
		expected string
	}{
		// Default paths
		{&types.SystemContext{}, "", "/run/containers/" + uid + "/auth.json"},
		{nil, "", "/run/containers/" + uid + "/auth.json"},
		// SystemContext overrides
		{&types.SystemContext{AuthFilePath: "/absolute/path"}, "", "/absolute/path"},
		{&types.SystemContext{RootForImplicitAbsolutePaths: "/prefix"}, "", "/prefix/run/containers/" + uid + "/auth.json"},
		// XDG_RUNTIME_DIR defined
		{nil, tmpDir, tmpDir + "/containers/auth.json"},
		{nil, tmpDir + "/thisdoesnotexist", ""},
	} {
		if c.xrd != "" {
			os.Setenv("XDG_RUNTIME_DIR", c.xrd)
		} else {
			os.Unsetenv("XDG_RUNTIME_DIR")
		}
		res, err := getPathToAuth(c.sys)
		if c.expected == "" {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, c.expected, res)
		}
	}
}

func TestGetAuth(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	tmpDir1, err := ioutil.TempDir("", "test_docker_client_get_auth")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("using temporary XDG_RUNTIME_DIR directory: %q", tmpDir1)
	// override XDG_RUNTIME_DIR
	os.Setenv("XDG_RUNTIME_DIR", tmpDir1)
	defer func() {
		err := os.RemoveAll(tmpDir1)
		if err != nil {
			t.Logf("failed to cleanup temporary home directory %q: %v", tmpDir1, err)
		}
		os.Setenv("XDG_RUNTIME_DIR", origXDG)
	}()

	origHomeDir := homedir.Get()
	tmpDir2, err := ioutil.TempDir("", "test_docker_client_get_auth")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("using temporary home directory: %q", tmpDir2)
	//override homedir
	os.Setenv(homedir.Key(), tmpDir2)
	defer func() {
		err := os.RemoveAll(tmpDir2)
		if err != nil {
			t.Logf("failed to cleanup temporary home directory %q: %v", tmpDir2, err)
		}
		os.Setenv(homedir.Key(), origHomeDir)
	}()

	configDir1 := filepath.Join(tmpDir1, "containers")
	if err := os.MkdirAll(configDir1, 0700); err != nil {
		t.Fatal(err)
	}
	configDir2 := filepath.Join(tmpDir2, ".docker")
	if err := os.MkdirAll(configDir2, 0700); err != nil {
		t.Fatal(err)
	}
	configPaths := [2]string{filepath.Join(configDir1, "auth.json"), filepath.Join(configDir2, "config.json")}

	for _, configPath := range configPaths {
		for _, tc := range []struct {
			name             string
			hostname         string
			path             string
			expectedUsername string
			expectedPassword string
			expectedError    error
			sys              *types.SystemContext
		}{
			{
				name:     "no auth config",
				hostname: "index.docker.io",
			},
			{
				name: "empty hostname",
				path: filepath.Join("testdata", "example.json"),
			},
			{
				name:             "match one",
				hostname:         "example.org",
				path:             filepath.Join("testdata", "example.json"),
				expectedUsername: "example",
				expectedPassword: "org",
			},
			{
				name:     "match none",
				hostname: "registry.example.org",
				path:     filepath.Join("testdata", "example.json"),
			},
			{
				name:             "match docker.io",
				hostname:         "docker.io",
				path:             filepath.Join("testdata", "full.json"),
				expectedUsername: "docker",
				expectedPassword: "io",
			},
			{
				name:             "match docker.io normalized",
				hostname:         "docker.io",
				path:             filepath.Join("testdata", "abnormal.json"),
				expectedUsername: "index",
				expectedPassword: "docker.io",
			},
			{
				name:             "normalize registry",
				hostname:         "https://example.org/v1",
				path:             filepath.Join("testdata", "full.json"),
				expectedUsername: "example",
				expectedPassword: "org",
			},
			{
				name:             "match localhost",
				hostname:         "http://localhost",
				path:             filepath.Join("testdata", "full.json"),
				expectedUsername: "local",
				expectedPassword: "host",
			},
			{
				name:             "match ip",
				hostname:         "10.10.30.45:5000",
				path:             filepath.Join("testdata", "full.json"),
				expectedUsername: "10.10",
				expectedPassword: "30.45-5000",
			},
			{
				name:             "match port",
				hostname:         "https://localhost:5000",
				path:             filepath.Join("testdata", "abnormal.json"),
				expectedUsername: "local",
				expectedPassword: "host-5000",
			},
			{
				name:             "use system context",
				hostname:         "example.org",
				path:             filepath.Join("testdata", "example.json"),
				expectedUsername: "foo",
				expectedPassword: "bar",
				sys: &types.SystemContext{
					DockerAuthConfig: &types.DockerAuthConfig{
						Username: "foo",
						Password: "bar",
					},
				},
			},
		} {
			if tc.path == "" {
				if err := os.RemoveAll(configPath); err != nil {
					t.Fatal(err)
				}
			}

			t.Run(tc.name, func(t *testing.T) {
				if tc.path != "" {
					contents, err := ioutil.ReadFile(tc.path)
					if err != nil {
						t.Fatal(err)
					}

					if err := ioutil.WriteFile(configPath, contents, 0640); err != nil {
						t.Fatal(err)
					}
				}

				var sys *types.SystemContext
				if tc.sys != nil {
					sys = tc.sys
				}
				username, password, err := GetAuthentication(sys, tc.hostname)
				assert.Equal(t, tc.expectedError, err)
				assert.Equal(t, tc.expectedUsername, username)
				assert.Equal(t, tc.expectedPassword, password)
			})
		}
	}
}

func TestGetAuthFromLegacyFile(t *testing.T) {
	origHomeDir := homedir.Get()
	tmpDir, err := ioutil.TempDir("", "test_docker_client_get_auth")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("using temporary home directory: %q", tmpDir)
	// override homedir
	os.Setenv(homedir.Key(), tmpDir)
	defer func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			t.Logf("failed to cleanup temporary home directory %q: %v", tmpDir, err)
		}
		os.Setenv(homedir.Key(), origHomeDir)
	}()

	configPath := filepath.Join(tmpDir, ".dockercfg")
	contents, err := ioutil.ReadFile(filepath.Join("testdata", "legacy.json"))
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name             string
		hostname         string
		expectedUsername string
		expectedPassword string
		expectedError    error
	}{
		{
			name:             "normalize registry",
			hostname:         "https://docker.io/v1",
			expectedUsername: "docker",
			expectedPassword: "io-legacy",
		},
		{
			name:             "ignore schema and path",
			hostname:         "http://index.docker.io/v1",
			expectedUsername: "docker",
			expectedPassword: "io-legacy",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ioutil.WriteFile(configPath, contents, 0640); err != nil {
				t.Fatal(err)
			}

			username, password, err := GetAuthentication(nil, tc.hostname)
			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedUsername, username)
			assert.Equal(t, tc.expectedPassword, password)
		})
	}
}

func TestGetAuthPreferNewConfig(t *testing.T) {
	origHomeDir := homedir.Get()
	tmpDir, err := ioutil.TempDir("", "test_docker_client_get_auth")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("using temporary home directory: %q", tmpDir)
	// override homedir
	os.Setenv(homedir.Key(), tmpDir)
	defer func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			t.Logf("failed to cleanup temporary home directory %q: %v", tmpDir, err)
		}
		os.Setenv(homedir.Key(), origHomeDir)
	}()

	configDir := filepath.Join(tmpDir, ".docker")
	if err := os.Mkdir(configDir, 0750); err != nil {
		t.Fatal(err)
	}

	for _, data := range []struct {
		source string
		target string
	}{
		{
			source: filepath.Join("testdata", "full.json"),
			target: filepath.Join(configDir, "config.json"),
		},
		{
			source: filepath.Join("testdata", "legacy.json"),
			target: filepath.Join(tmpDir, ".dockercfg"),
		},
	} {
		contents, err := ioutil.ReadFile(data.source)
		if err != nil {
			t.Fatal(err)
		}

		if err := ioutil.WriteFile(data.target, contents, 0640); err != nil {
			t.Fatal(err)
		}
	}

	username, password, err := GetAuthentication(nil, "docker.io")
	assert.Equal(t, nil, err)
	assert.Equal(t, "docker", username)
	assert.Equal(t, "io", password)
}

func TestGetAuthFailsOnBadInput(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	tmpDir1, err := ioutil.TempDir("", "test_docker_client_get_auth")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("using temporary XDG_RUNTIME_DIR directory: %q", tmpDir1)
	// override homedir
	os.Setenv("XDG_RUNTIME_DIR", tmpDir1)
	defer func() {
		err := os.RemoveAll(tmpDir1)
		if err != nil {
			t.Logf("failed to cleanup temporary home directory %q: %v", tmpDir1, err)
		}
		os.Setenv("XDG_RUNTIME_DIR", origXDG)
	}()

	origHomeDir := homedir.Get()
	tmpDir2, err := ioutil.TempDir("", "test_docker_client_get_auth")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("using temporary home directory: %q", tmpDir2)
	// override homedir
	os.Setenv(homedir.Key(), tmpDir2)
	defer func() {
		err := os.RemoveAll(tmpDir2)
		if err != nil {
			t.Logf("failed to cleanup temporary home directory %q: %v", tmpDir2, err)
		}
		os.Setenv(homedir.Key(), origHomeDir)
	}()

	configDir := filepath.Join(tmpDir1, "containers")
	if err := os.Mkdir(configDir, 0750); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "auth.json")

	// no config file present
	username, password, err := GetAuthentication(nil, "index.docker.io")
	if err != nil {
		t.Fatalf("got unexpected error: %#+v", err)
	}
	if len(username) > 0 || len(password) > 0 {
		t.Fatalf("got unexpected not empty username/password: %q/%q", username, password)
	}

	if err := ioutil.WriteFile(configPath, []byte("Json rocks! Unless it doesn't."), 0640); err != nil {
		t.Fatalf("failed to write file %q: %v", configPath, err)
	}
	username, password, err = GetAuthentication(nil, "index.docker.io")
	if err == nil {
		t.Fatalf("got unexpected non-error: username=%q, password=%q", username, password)
	}
	if _, ok := errors.Cause(err).(*json.SyntaxError); !ok {
		t.Fatalf("expected JSON syntax error, not: %#+v", err)
	}

	// remove the invalid config file
	os.RemoveAll(configPath)
	// no config file present
	username, password, err = GetAuthentication(nil, "index.docker.io")
	if err != nil {
		t.Fatalf("got unexpected error: %#+v", err)
	}
	if len(username) > 0 || len(password) > 0 {
		t.Fatalf("got unexpected not empty username/password: %q/%q", username, password)
	}

	configPath = filepath.Join(tmpDir2, ".dockercfg")
	if err := ioutil.WriteFile(configPath, []byte("I'm certainly not a json string."), 0640); err != nil {
		t.Fatalf("failed to write file %q: %v", configPath, err)
	}
	username, password, err = GetAuthentication(nil, "index.docker.io")
	if err == nil {
		t.Fatalf("got unexpected non-error: username=%q, password=%q", username, password)
	}
	if _, ok := errors.Cause(err).(*json.SyntaxError); !ok {
		t.Fatalf("expected JSON syntax error, not: %#+v", err)
	}
}
