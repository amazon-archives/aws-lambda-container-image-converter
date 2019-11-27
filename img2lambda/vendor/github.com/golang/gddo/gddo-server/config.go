package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/golang/gddo/log"
)

const (
	gaeProjectEnvVar = "GCLOUD_PROJECT"
	gaAccountEnvVar  = "GA_ACCOUNT"

	userAgentEnvVar          = "USER_AGENT"
	githubTokenEnvVar        = "GITHUB_TOKEN"
	githubClientIDEnvVar     = "GITHUB_CLIENT_ID"
	githubClientSecretEnvVar = "GITHUB_CLIENT_SECRET"
)

const (
	// Server Config
	ConfigProject           = "project"
	ConfigTrustProxyHeaders = "trust_proxy_headers"
	ConfigBindAddress       = "http"
	ConfigAssetsDir         = "assets"
	ConfigRobotThreshold    = "robot"
	ConfigGCELogName        = "gce_log_name"

	// Database Config
	ConfigDBServer      = "db-server"
	ConfigDBIdleTimeout = "db-idle-timeout"
	ConfigDBLog         = "db-log"
	ConfigGAERemoteAPI  = "remoteapi-endpoint"

	// Display Config
	ConfigSidebar        = "sidebar"
	ConfigSourcegraphURL = "sourcegraph_url"
	ConfigDefaultGOOS    = "default_goos"
	ConfigGAAccount      = "ga_account"

	// Crawl Config
	ConfigMaxAge          = "max_age"
	ConfigGetTimeout      = "get_timeout"
	ConfigFirstGetTimeout = "first_get_timeout"
	ConfigGithubInterval  = "github_interval"
	ConfigCrawlInterval   = "crawl_interval"
	ConfigDialTimeout     = "dial_timeout"
	ConfigRequestTimeout  = "request_timeout"
	ConfigMemcacheAddr    = "memcache_addr"

	// Trace Config
	ConfigTraceSamplerFraction = "trace_fraction"
	ConfigTraceSamplerMaxQPS   = "trace_max_qps"

	// Outbound HTTP Config
	ConfigUserAgent          = "user_agent"
	ConfigGithubToken        = "github_token"
	ConfigGithubClientID     = "github_client_id"
	ConfigGithubClientSecret = "github_client_secret"

	// Pub/Sub Config
	ConfigCrawlPubSubTopic = "crawl-events"
)

func loadConfig(ctx context.Context, args []string) (*viper.Viper, error) {
	v := viper.New()
	// Gather information from execution environment.
	if os.Getenv(gaeProjectEnvVar) != "" {
		v.Set("on_appengine", true)
	} else {
		v.Set("on_appengine", false)
	}
	if metadata.OnGCE() {
		gceProjectAttributeDefault(ctx, v, ConfigGAAccount, "ga-account")
		gceProjectAttributeDefault(ctx, v, ConfigGCELogName, "gce-log-name")
		gceProjectAttributeDefault(ctx, v, ConfigUserAgent, "user-agent")
		gceProjectAttributeDefault(ctx, v, ConfigGithubToken, "github-token")
		gceProjectAttributeDefault(ctx, v, ConfigGithubClientID, "github-client-id")
		gceProjectAttributeDefault(ctx, v, ConfigGithubClientSecret, "github-client-secret")
		if id, err := metadata.ProjectID(); err != nil {
			log.Warn(ctx, "failed to retrieve project ID", "error", err)
		} else {
			viper.SetDefault(ConfigProject, id)
		}
	}

	// Setup command line flags
	flags := buildFlags()
	if err := flags.Parse(args); err != nil {
		return nil, err
	}
	if err := v.BindPFlags(flags); err != nil {
		return nil, err
	}

	// Also fetch from environment
	v.SetEnvPrefix("gddo")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()
	v.BindEnv(ConfigProject, gaeProjectEnvVar)
	v.BindEnv(ConfigGAAccount, gaAccountEnvVar)
	v.BindEnv(ConfigUserAgent, userAgentEnvVar)
	v.BindEnv(ConfigGithubToken, githubTokenEnvVar)
	v.BindEnv(ConfigGithubClientID, githubClientIDEnvVar)
	v.BindEnv(ConfigGithubClientSecret, githubClientSecretEnvVar)

	// Read from config.
	if err := readViperConfig(ctx, v); err != nil {
		return nil, err
	}

	// Set defaults based on other configs
	setDefaults(v)

	log.Debug(ctx, "config values loaded", "values", v.AllSettings())
	return v, nil
}

func gceProjectAttributeDefault(ctx context.Context, v *viper.Viper, cfg, attr string) {
	val, err := metadata.ProjectAttributeValue(attr)
	if err != nil {
		if _, undef := err.(metadata.NotDefinedError); !undef {
			log.Warn(ctx, "failed to query metadata", "key", attr, "error", err)
		}
		return
	}
	v.SetDefault(cfg, val)
}

// setDefaults sets defaults for configuration options that depend on other
// configuration options. This allows for smart defaults but allows for
// overrides.
func setDefaults(v *viper.Viper) {
	// ConfigGAERemoteAPI is based on project.
	project := v.GetString(ConfigProject)
	if project != "" {
		defaultEndpoint := fmt.Sprintf("serviceproxy-dot-%s.appspot.com", project)
		v.SetDefault(ConfigGAERemoteAPI, defaultEndpoint)
	}
}

func buildFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("default", pflag.ContinueOnError)

	flags.StringP("config", "c", "", "path to motd config file")
	flags.String(ConfigProject, "", "Google Cloud Platform project used for Google services")
	flags.Float64(ConfigRobotThreshold, 100, "Request counter threshold for robots.")
	flags.String(ConfigAssetsDir, filepath.Join(defaultBase("github.com/golang/gddo/gddo-server"), "assets"), "Base directory for templates and static files.")
	flags.Duration(ConfigGetTimeout, 8*time.Second, "Time to wait for package update from the VCS.")
	flags.Duration(ConfigFirstGetTimeout, 5*time.Second, "Time to wait for first fetch of package from the VCS.")
	flags.Duration(ConfigMaxAge, 24*time.Hour, "Update package documents older than this age.")
	flags.String(ConfigBindAddress, ":8080", "Listen for HTTP connections on this address.")
	flags.Bool(ConfigSidebar, false, "Enable package page sidebar.")
	flags.String(ConfigDefaultGOOS, "", "Default GOOS to use when building package documents.")
	flags.Bool(ConfigTrustProxyHeaders, false, "If enabled, identify the remote address of the request using X-Real-Ip in header.")
	flags.String(ConfigSourcegraphURL, "https://sourcegraph.com", "Link to global uses on Sourcegraph based at this URL (no need for trailing slash).")
	flags.Duration(ConfigGithubInterval, 0, "Github updates crawler sleeps for this duration between fetches. Zero disables the crawler.")
	flags.Duration(ConfigCrawlInterval, 0, "Package updater sleeps for this duration between package updates. Zero disables updates.")
	flags.Duration(ConfigDialTimeout, 5*time.Second, "Timeout for dialing an HTTP connection.")
	flags.Duration(ConfigRequestTimeout, 20*time.Second, "Time out for roundtripping an HTTP request.")
	flags.String(ConfigDBServer, "redis://127.0.0.1:6379", "URI of Redis server.")
	flags.Duration(ConfigDBIdleTimeout, 250*time.Second, "Close Redis connections after remaining idle for this duration.")
	flags.Bool(ConfigDBLog, false, "Log database commands")
	flags.String(ConfigMemcacheAddr, "", "Address in the format host:port gddo uses to point to the memcache backend.")
	flags.String(ConfigGAERemoteAPI, "", "Remoteapi endpoint for App Engine Search. Defaults to serviceproxy-dot-${project}.appspot.com.")
	flags.Float64(ConfigTraceSamplerFraction, 0.1, "Fraction of the requests sampled by the trace API.")
	flags.Float64(ConfigTraceSamplerMaxQPS, 5, "Max number of requests sampled every second by the trace API.")

	return flags
}

// readViperConfig finds and then parses a config file. It will return
// an error if the config file was specified or could not parse.
// Otherwise it will only warn that it failed to load the config.
func readViperConfig(ctx context.Context, v *viper.Viper) error {
	v.AddConfigPath(".")
	v.AddConfigPath("/etc")
	v.SetConfigName("gddo")
	if v.GetString("config") != "" {
		v.SetConfigFile(v.GetString("config"))
	}

	if err := v.ReadInConfig(); err != nil {
		// If a config exists but could not be parsed, we should bail.
		if _, ok := err.(viper.ConfigParseError); ok {
			return fmt.Errorf("parse config: %v", err)
		}

		// If the user specified a config file location in flags or env and
		// we failed to load it, we should bail. If not, it is just a warning.
		if v.GetString("config") != "" {
			return fmt.Errorf("load config: %v", err)
		}
		log.Warn(ctx, "failed to load configuration file", "error", err)
		return nil
	}
	log.Info(ctx, "loaded configuration file successfully", "path", v.ConfigFileUsed())
	return nil
}
