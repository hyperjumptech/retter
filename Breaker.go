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
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
	"net/http"
	"time"
)

var (
	breakerLog = logrus.WithFields(logrus.Fields{
		"module": "GoBreaker",
		"file":   "Breaker.go",
	})

	// PathBreakers is a map of string to gobreaker.CircuitBreaker.
	// The string key is a full path + session key information.
	// This makes each user's accessible path is circuit breaked.
	PathBreakers = make(map[string]*gobreaker.CircuitBreaker)
)

// GetBreakerSettingForRequest will create a grobreaker.Setting for each created CircuitBreaker.
func GetBreakerSettingForRequest(req *http.Request) gobreaker.Settings {
	completePath := req.URL.Path
	if len(req.URL.RawQuery) > 0 {
		completePath = fmt.Sprintf("%s?%s", completePath, req.URL.RawQuery)
	}
	return gobreaker.Settings{
		Name:        completePath,
		MaxRequests: 1,
		Interval:    10 * time.Second,
		Timeout:     0,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			done := counts.TotalFailures + counts.TotalSuccesses
			if done > 0 && counts.Requests > 4 {
				breakerLog.Tracef("[%s] ready to trip. totalFail %d of %d", completePath, counts.TotalFailures, done)
				failRate := float64(counts.TotalFailures) / float64(done)
				return failRate > Config.GetFloat(FailureRate)
			}
			return int(counts.ConsecutiveFailures) > Config.GetInt(ConsecutiveFail)
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			breakerLog.Tracef("[%s] changed state from %s to %s", completePath, from.String(), to.String())
		},
	}
}

// GetBreakerForRequest returns a CircuitBreaker to be use for circuit breaking
// each particular request.
func GetBreakerForRequest(req *http.Request) *gobreaker.CircuitBreaker {
	completePath := req.URL.Path
	if len(req.URL.RawQuery) > 0 {
		completePath = fmt.Sprintf("%s?%s", completePath, req.URL.RawQuery)
	}
	if b, ok := PathBreakers[completePath]; ok {
		return b
	}
	newBreaker := gobreaker.NewCircuitBreaker(GetBreakerSettingForRequest(req))
	PathBreakers[completePath] = newBreaker
	return newBreaker
}
