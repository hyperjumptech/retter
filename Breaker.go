package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
	"net/http"
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
		MaxRequests: 0,
		Interval:    0,
		Timeout:     0,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			breakerLog.Infof("[%s] changed state from %s to %s", completePath, from.String(), to.String())
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
