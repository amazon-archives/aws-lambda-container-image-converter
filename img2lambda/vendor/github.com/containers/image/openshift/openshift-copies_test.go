package openshift

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixtureKubeConfigPath = "testdata/admin.kubeconfig"

// These are only smoke tests based on the skopeo integration test cluster. Error handling, non-trivial configuration merging,
// and any other situations are not currently covered.

// Set up KUBECONFIG to point at the fixture, and return a handler to clean it up.
// Callers MUST NOT call testing.T.Parallel().
func setupKubeConfigForSerialTest() func() {
	// Environment is per-process, so this looks very unsafe; actually it seems fine because tests are not
	// run in parallel unless they opt in by calling t.Parallel().  So don’t do that.
	oldKC, hasKC := os.LookupEnv("KUBECONFIG")
	cleanup := func() {
		if hasKC {
			os.Setenv("KUBECONFIG", oldKC)
		} else {
			os.Unsetenv("KUBECONFIG")
		}
	}
	os.Setenv("KUBECONFIG", fixtureKubeConfigPath)
	return cleanup
}

func TestClientConfigLoadingRules(t *testing.T) {
	cleanup := setupKubeConfigForSerialTest()
	defer cleanup()

	rules := newOpenShiftClientConfigLoadingRules()
	res, err := rules.Load()
	require.NoError(t, err)
	expected := clientcmdConfig{
		Clusters: clustersMap{
			"172-17-0-2:8443": &clientcmdCluster{
				LocationOfOrigin:         fixtureKubeConfigPath,
				Server:                   "https://172.17.0.2:8443",
				CertificateAuthorityData: []byte("Cluster CA"),
			},
		},
		AuthInfos: authInfosMap{
			"system:admin/172-17-0-2:8443": &clientcmdAuthInfo{
				LocationOfOrigin:      fixtureKubeConfigPath,
				ClientCertificateData: []byte("Client cert"),
				ClientKeyData:         []byte("Client key"),
			},
		},
		Contexts: contextsMap{
			"default/172-17-0-2:8443/system:admin": &clientcmdContext{
				LocationOfOrigin: fixtureKubeConfigPath,
				Cluster:          "172-17-0-2:8443",
				AuthInfo:         "system:admin/172-17-0-2:8443",
				Namespace:        "default",
			},
		},
		CurrentContext: "default/172-17-0-2:8443/system:admin",
	}
	assert.Equal(t, &expected, res)
}

func TestDirectClientConfig(t *testing.T) {
	cleanup := setupKubeConfigForSerialTest()
	defer cleanup()

	rules := newOpenShiftClientConfigLoadingRules()
	config, err := rules.Load()
	require.NoError(t, err)

	direct := newNonInteractiveClientConfig(*config)
	res, err := direct.ClientConfig()
	require.NoError(t, err)
	assert.Equal(t, &restConfig{
		Host: "https://172.17.0.2:8443",
		restTLSClientConfig: restTLSClientConfig{
			CertData: []byte("Client cert"),
			KeyData:  []byte("Client key"),
			CAData:   []byte("Cluster CA"),
		},
	}, res)
}

func TestDeferredLoadingClientConfig(t *testing.T) {
	cleanup := setupKubeConfigForSerialTest()
	defer cleanup()

	rules := newOpenShiftClientConfigLoadingRules()
	deferred := newNonInteractiveDeferredLoadingClientConfig(rules)
	res, err := deferred.ClientConfig()
	require.NoError(t, err)
	assert.Equal(t, &restConfig{
		Host: "https://172.17.0.2:8443",
		restTLSClientConfig: restTLSClientConfig{
			CertData: []byte("Client cert"),
			KeyData:  []byte("Client key"),
			CAData:   []byte("Cluster CA"),
		},
	}, res)
}

func TestDefaultClientConfig(t *testing.T) {
	cleanup := setupKubeConfigForSerialTest()
	defer cleanup()

	config := defaultClientConfig()
	res, err := config.ClientConfig()
	require.NoError(t, err)
	assert.Equal(t, &restConfig{
		Host: "https://172.17.0.2:8443",
		restTLSClientConfig: restTLSClientConfig{
			CertData: []byte("Client cert"),
			KeyData:  []byte("Client key"),
			CAData:   []byte("Cluster CA"),
		},
	}, res)
}
