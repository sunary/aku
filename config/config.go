package config

import (
	"bytes"
	_ "embed"
	"strings"

	"github.com/spf13/viper"
	"github.com/sunary/aku/loging"
	"go.uber.org/zap"
)

var (
	ll = loging.New()
)

//go:embed default.yaml
var defaultConfig []byte

type Config struct {
	Http              *HttpConfig `json:"http" mapstructure:"http" yaml:"http"`
	Grpc              *GrpcConfig `json:"grpc" mapstructure:"grpc" yaml:"grpc"`
	IpForwardedHeader string      `json:"ip_forwarded_header" mapstructure:"ip_forwarded_header" yaml:"ip_forwarded_header"`
}

type HttpConfig struct {
	Port      int         `json:"port" mapstructure:"port" yaml:"port"`
	Timeout   int64       `json:"timeout" mapstructure:"timeout" yaml:"timeout"`
	HealthURI string      `json:"health_uri" mapstructure:"health_uri" yaml:"health_uri"`
	RouteMaps []HttpRoute `json:"route_maps" mapstructure:"route_maps" yaml:"route_maps"`
}

type HttpRoute struct {
	Name          string    `json:"name" mapstructure:"name" yaml:"name"`
	Host          string    `json:"host" mapstructure:"host" yaml:"host"`
	OverridePath  string    `json:"override_path" mapstructure:"override_path" yaml:"override_path"`
	UpstreamPath  string    `json:"upstream_path" mapstructure:"upstream_path" yaml:"upstream_path"`
	Plugins       []string  `json:"plugins" mapstructure:"plugins" yaml:"plugins"`
	Cors          *HttpCors `json:"cors" mapstructure:"cors" yaml:"cors"`
	IpRestriction []string  `json:"ip_restriction" mapstructure:"ip_restriction" yaml:"ip_restriction"`
}

type HttpCors struct {
	Origins           []string `json:"origins" mapstructure:"origins" yaml:"origins"`
	Methods           []string `json:"methods" mapstructure:"methods" yaml:"methods"`
	Headers           []string `json:"headers" mapstructure:"headers" yaml:"headers"`
	Credentials       bool     `json:"credentials" mapstructure:"credentials" yaml:"credentials"`
	PreflightContinue bool     `json:"preflight_continue" mapstructure:"preflight_continue" yaml:"preflight_continue"`
}

type GrpcConfig struct {
	Port           int          `json:"port" mapstructure:"port" yaml:"port"`
	KeepConnection bool         `json:"keep_connection" mapstructure:"keep_connection" yaml:"keep_connection"`
	Timeout        int64        `json:"timeout" mapstructure:"timeout" yaml:"timeout"`
	MethodMaps     []GrpcMethod `json:"method_maps" mapstructure:"method_maps" yaml:"method_maps"`
}

type GrpcMethod struct {
	Name          string   `json:"name" mapstructure:"name" yaml:"name"`
	Host          string   `json:"host" mapstructure:"host" yaml:"host"`
	ProtoService  string   `json:"proto_service" mapstructure:"proto_service" yaml:"proto_service"`
	Allow         []string `json:"allow" mapstructure:"allow" yaml:"allow"`
	Disallow      []string `json:"disallow" mapstructure:"disallow" yaml:"disallow"`
	Plugins       []string `json:"plugins" mapstructure:"plugins" yaml:"plugins"`
	IpRestriction []string `json:"ip_restriction" mapstructure:"ip_restriction" yaml:"ip_restriction"`
}

func Load() *Config {
	var cfg = &Config{}

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(bytes.NewBuffer(defaultConfig))
	if err != nil {
		ll.Fatal("Failed to read viper config", zap.String("err", err.Error()))
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	viper.AutomaticEnv()

	err = viper.Unmarshal(&cfg)
	if err != nil {
		ll.Fatal("Failed to unmarshal config", zap.String("err", err.Error()))
	}

	ll.Info("Config loaded", zap.Any("config", cfg))
	return cfg
}
