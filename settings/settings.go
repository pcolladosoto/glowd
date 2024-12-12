package settings

import (
	"fmt"
	"log/slog"
	"strings"

	glowdTypes "github.com/scitags/flowd-go/types"

	"github.com/scitags/flowd-go/backends/ebpf"
	"github.com/scitags/flowd-go/backends/firefly"
	"github.com/scitags/flowd-go/netlink"
	"github.com/scitags/flowd-go/plugins/api"
	"github.com/scitags/flowd-go/plugins/np"

	"github.com/spf13/viper"
)

type PluginConfigurations struct {
	NamedPipe np.NamedPipePlugin
	Api       api.ApiPlugin
	Netlink   netlink.NetlinkPlugin
}

type BackendConfigurations struct {
	Ebpf    ebpf.EbpfBackend
	Firefly firefly.FireflyBackend
}

type Configuration struct {
	General  GeneralConfiguration
	Plugins  []glowdTypes.Plugin
	Backends []glowdTypes.Backend
}

type defaultConfiguration map[string]interface{}

var (
	Defaults = map[string]defaultConfiguration{
		"": {
			"pidPath":     "/var/run/flowd-go.pid",
			"workDir":     "/var/cache/flowd-go",
			"stunServers": nil,
		},
	}

	pluginDefaults = map[string]defaultConfiguration{
		"namedPipe": np.Defaults,
		"api":       api.Defaults,
		"netlink":   netlink.Defaults,
	}

	backendDefaults = map[string]defaultConfiguration{
		"ebpf":    ebpf.Defaults,
		"firefly": firefly.Defaults,
	}
)

// Sample config ripped from flowd
// CONFIG_PATH = '/etc/flowd/flowd.cfg'
// PID_FILE = '/var/run/flowd.pid'
// WORK_DIR = '/var/cache/flowd'
// DEFAULT_BACKEND = 'udp_firefly'
// NP_API_FILE = '/var/run/flowd'
// UDP_FIREFLY_PORT = 10514
// IP4_DISCOVERY = ('10.255.255.255', 1)
// IP6_DISCOVERY = ('fc00::', 1)
// STUN_SERVERS = [('stun.l.google.com', 19305), ('stun.services.mozilla.org', 3478)]
// NETSTAT_TIMEOUT = 2
// FIREFLY_LISTENER_HOST = "0.0.0.0"
// FIREFLY_LISTENER_PORT = 10514
// NETLINK_TIMEOUT = 2
// PROMETHEUS_SRV_PORT = 9000
// SS_PATH = '/usr/sbin/ss'

type GeneralConfiguration struct {
	PIDPath     string
	WorkDir     string
	StunServers []string
}

func populateDefaults(conf *viper.Viper, defs map[string]defaultConfiguration) {
	for k, v := range defs {
		for kk, vv := range v {
			// If setting the general defaults...
			if k == "" {
				conf.SetDefault(kk, vv)
				continue
			}
			conf.SetDefault(fmt.Sprintf("%s.%s", k, kk), vv)
		}
	}
}

func populateBP(conf *viper.Viper, path string, defaults map[string]defaultConfiguration, unmarshalTarget interface{}) ([]string, error) {
	subConf := conf.Sub(path)
	if subConf == nil {
		return nil, fmt.Errorf("no %s configured: you need at least one", path)
	}

	// Get a hold of the configured plugins/backends before setting the defaults. Doing
	// so will always trigger IsSet()!
	configuredKeys := []string{}
	for k := range defaults {
		if subConf.IsSet(k) {
			configuredKeys = append(configuredKeys, k)
		}
	}

	// The defaults are not propagated when calling Sub()...
	populateDefaults(subConf, defaults)

	if err := subConf.Unmarshal(unmarshalTarget); err != nil {
		return nil, err
	}

	return configuredKeys, nil
}

func populatePluginSlice(pConf PluginConfigurations, configured []string) ([]glowdTypes.Plugin, error) {
	plugins := []glowdTypes.Plugin{}
	for _, c := range configured {
		switch strings.ToLower(c) {
		case strings.ToLower("namedPipe"):
			plugins = append(plugins, &pConf.NamedPipe)
		case strings.ToLower("api"):
			plugins = append(plugins, &pConf.Api)
		case strings.ToLower("netlink"):
			plugins = append(plugins, &pConf.Netlink)
		default:
			return nil, fmt.Errorf("plugin type %q is not recognized", c)
		}
	}
	return plugins, nil
}

func populateBackendSlice(bConf BackendConfigurations, configured []string) ([]glowdTypes.Backend, error) {
	backends := []glowdTypes.Backend{}
	for _, c := range configured {
		switch strings.ToLower(c) {
		case strings.ToLower("ebpf"):
			backends = append(backends, &bConf.Ebpf)
		case strings.ToLower("firefly"):
			backends = append(backends, &bConf.Firefly)
		default:
			return nil, fmt.Errorf("backend type %q is not recognized", c)
		}
	}
	return backends, nil
}

func ReadConf(confFile string) (*Configuration, error) {
	conf := viper.New()
	conf.SetConfigFile(confFile)

	if err := conf.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading the configuration: %w", err)
	}

	pConf := PluginConfigurations{}
	configuredPlugins, err := populateBP(conf, "plugins", pluginDefaults, &pConf)
	if err != nil {
		return nil, fmt.Errorf("couldn't unmarshal the plugin configuration: %w", err)
	}
	plugins, err := populatePluginSlice(pConf, configuredPlugins)
	if err != nil {
		return nil, fmt.Errorf("error populating the plugin slice: %w", err)
	}

	bConf := BackendConfigurations{}
	configuredBackends, err := populateBP(conf, "backends", backendDefaults, &bConf)
	if err != nil {
		return nil, fmt.Errorf("couldn't unmarshal the plugin configuration: %w", err)
	}
	backends, err := populateBackendSlice(bConf, configuredBackends)
	if err != nil {
		return nil, fmt.Errorf("error populating the backend slice: %w", err)
	}

	populateDefaults(conf, Defaults)
	gConf := GeneralConfiguration{}
	if err := conf.Unmarshal(&gConf); err != nil {
		return nil, fmt.Errorf("error unmarshaling the general configuration: %w", err)
	}

	slog.Debug("loaded general configuration", "gConf", gConf)
	slog.Debug("loaded plugin configuration", "pConf", pConf)
	slog.Debug("loaded backend configuration", "bConf", bConf)

	return &Configuration{
		General:  gConf,
		Plugins:  plugins,
		Backends: backends,
	}, nil
}
