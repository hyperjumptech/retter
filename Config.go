package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"reflect"
	"strconv"
	"strings"
)

const (
	CACHE_TTL           = "cache.ttl"
	CACHE_DETECTSESSION = "cache.detectsession"
	BACKEND_URL         = "backend.baseurl"
	SERVER_LISTEN       = "server.listen"
)

var (
	configLog = logrus.WithFields(logrus.Fields{
		"module": "Configuration",
		"file":   "Config.go",
	})
	Config = Configuration{
		CACHE_TTL:                  "60",    // time to live in seconds
		CACHE_DETECTSESSION:        "false", // always account session cookie in the cache
		BACKEND_URL:                "http://localhost:8088",
		SERVER_LISTEN:              ":8089",
		"server.timeout.write":     "15 seconds",
		"server.timeout.read":      "15 seconds",
		"server.timeout.idle":      "60 seconds",
		"server.timeout.graceshut": "15 seconds",
	}
)

func init() {
	viper.SetEnvPrefix("retter")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	for k := range Config {
		err := viper.BindEnv(k)
		if err != nil {
			configLog.Errorf("Failed to bind env \"%s\" into configuration. Got %s", k, err)
		}
	}
}

type Configuration map[string]string

func (c Configuration) GetString(key string) string {
	if valStr, ok := c[key]; ok {
		ret := viper.GetString(key)
		if len(ret) == 0 {
			return valStr
		}
		return ret
	}
	return ""
}

func (c Configuration) GetInt(key string) int {
	if valStr, ok := c[key]; ok {
		ret := viper.GetInt(key)
		if reflect.ValueOf(ret).IsZero() {
			i, err := strconv.Atoi(valStr)
			if err != nil {
				return 0
			}
			return i
		}
		return ret
	}
	return 0
}

func (c Configuration) GetFloat(key string) float64 {
	if valStr, ok := c[key]; ok {
		ret := viper.GetFloat64(key)
		if reflect.ValueOf(ret).IsZero() {
			f, err := strconv.ParseFloat(valStr, 64)
			if err != nil {
				return 0
			}
			return f
		}
		return ret
	}
	return 0
}

func (c Configuration) GetBoolean(key string) bool {
	if valStr, ok := c[key]; ok {
		ret := viper.GetBool(key)
		if reflect.ValueOf(ret).IsZero() {
			b, err := strconv.ParseBool(valStr)
			if err != nil {
				return false
			}
			return b
		}
		return ret
	}
	return false
}
