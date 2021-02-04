/*-----------------------------------------------------------------------------------
  --  RETTER                                                                       --
  --  Copyright (C) 2021  RETTER's Contributors                                    --
  --                                                                               --
  --  This program is free software: you can redistribute it and/or modify         --
  --  it under the terms of the GNU Affero General Public License as published     --
  --  by the Free Software Foundation, either version 3 of the License, or         --
  --  (at your option) any later version.                                          --
  --                                                                               --
  --  This program is distributed in the hope that it will be useful,              --
  --  but WITHOUT ANY WARRANTY; without even the implied warranty of               --
  --  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the                --
  --  GNU Affero General Public License for more details.                          --
  --                                                                               --
  --  You should have received a copy of the GNU Affero General Public License     --
  --  along with this program.  If not, see <https:   -- www.gnu.org/licenses/>.   --
  -----------------------------------------------------------------------------------*/

package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"reflect"
	"strconv"
	"strings"
)

const (
	// CacheTTL is key config for number of TTL in second
	CacheTTL = "cache.ttl"

	// CacheDetectQuery is key config for specifying whether to include session detection or not
	CacheDetectQuery = "cache.detect.query"

	// CacheDetectSession is key config for specifying whether to include session detection or not
	CacheDetectSession = "cache.detect.session"

	// BackendURL is key config for the base URL to call to backend
	BackendURL = "backend.baseurl"

	// ServerListen is key config for the server listening setting (bind host and port)
	ServerListen = "server.listen"

	// FailureRate is key config for the failure rate detection in the CircuitBreaker.
	// If the request to backend has reached this failure rate, circuit will open.
	// The fail rate will reset every 10 second
	FailureRate = "breaker.fail.rate"

	// ConsecutiveFail is key config for the number of consecutive backend http call fails.
	ConsecutiveFail = "breaker.consecutive.fail"
)

var (
	configLog = logrus.WithFields(logrus.Fields{
		"module": "Configuration",
		"file":   "Config.go",
	})

	// Config is the configuration instance
	Config = Configuration{
		CacheTTL:                   "60",    // time to live in seconds
		CacheDetectSession:         "false", // always account session cookie in the cache
		CacheDetectQuery:           "true",  // always account request URL query in the cache
		BackendURL:                 "http://localhost:8088",
		ServerListen:               ":8089",
		"server.timeout.write":     "15 seconds",
		"server.timeout.read":      "15 seconds",
		"server.timeout.idle":      "60 seconds",
		"server.timeout.graceshut": "15 seconds",
		FailureRate:                "0.66",
		ConsecutiveFail:            "5",
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

// Configuration configuration is a simple string to string map to store RETTER configuration.
// Although its a string to string map, developer should not access this map directly to obtain configuration value.
// Use GetString, GetInt, GetBool or GetFloat function as it uses Viper to sync configuration with environment variable.
type Configuration map[string]string

// GetString will return string configuration value of a string key
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

// GetInt will return integer configuration value of a string key
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

// GetFloat will return float64 configuration value of a string key
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

// GetBoolean will return bool configuration value of a string key
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
