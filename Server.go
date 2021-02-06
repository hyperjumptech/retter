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
	"github.com/hyperjumptech/jiffy"
	"github.com/hyperjumptech/retter/cache"
	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"runtime"
	"strings"
	"time"
)

const (
	// RetterStatusBackendTimeout is a custom HTTP response code if
	// the http client timed out while trying to call the backend server
	RetterStatusBackendTimeout = 1000
)

var (
	serverLog = logrus.WithFields(logrus.Fields{
		"module": "RetterHTTPHandler",
		"file":   "Server.go",
	})

	lastKnownSuccess = make(map[string]HTTPTransaction)

	// ServerStarTime is a variable to store server start time.
	ServerStarTime time.Time

	// RequestCount to store total served request, with exception to /health health check path.
	RequestCount uint16

	// TotalResponseTime is a total response time in millisecond been recorded by this RETTER server
	TotalResponseTime uint64

	// SlowestResponseTime is the number of milliseconds of the slowest response time.
	SlowestResponseTime uint64

	// FastestResponseTime is the number of milliseconds of the fastest response time.
	FastestResponseTime uint64
)

func init() {
	ServerStarTime = time.Now()
}

// NewRetterHTTPHandler create new http.Handler for this Retter server
func NewRetterHTTPHandler() http.Handler {
	l := Config.GetString(ServerListen)
	if l[0:1] == ":" {
		l = "http://0.0.0.0" + l
	}
	logrus.Infof("This RETTER instance will forwards GET request...\n\tFrom : %s/*\n\tTo   : %s/*\n", l, Config.GetString(BackendURL))
	logrus.Infof("URL Query Detect         : %s\n", Config.GetString(CacheDetectQuery))
	logrus.Infof("URL Session Detect       : %s\n", Config.GetString(CacheDetectSession))
	return &RetterHTTPHandler{
		BackendBaseURL: Config.GetString(BackendURL),
	}
}

// RetterHTTPHandler an implementation of http.Handler
type RetterHTTPHandler struct {
	BackendBaseURL string
}

// ServeHTTP is the handling method of incoming HTTP request and response
func (rhh *RetterHTTPHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if strings.ToUpper(req.Method) == "GET" && req.URL.Path == "/health" {
		res.Header().Add("Content-Type", "application/json")
		res.WriteHeader(http.StatusOK)

		uptime := jiffy.DescribeDuration(time.Since(ServerStarTime), jiffy.NewWant())
		cacheCount := cache.CacheSize()
		timerCount := cache.TimerSize()
		breakerCount := len(PathBreakers)

		AverageResponseTime := float64(TotalResponseTime) / float64(RequestCount)

		memStat := &runtime.MemStats{}
		runtime.ReadMemStats(memStat)

		body := fmt.Sprintf("{\"status\":\"OK\", "+
			"\"server-uptime\": \"%s\", "+
			"\"cache-count\":%d, "+
			"\"ttl-timer-count\":%d, "+
			"\"breaker-count\":%d, "+
			"\"total-request-served\":%d, "+
			"\"total-response-time-ms\":%d, "+
			"\"average-response-time-ms\":%f,"+
			"\"slowest-response-time-ms\":%d,"+
			"\"fastest-response-time-ms\":%d,"+
			"\"memory\":{"+
			"\"sys-memory-byte\":%d, "+
			"\"alloc-memory-byte\":%d, "+
			"\"total-alloc-memory-byte\":%d"+
			"}}", uptime, cacheCount, timerCount, breakerCount,
			RequestCount, TotalResponseTime, AverageResponseTime,
			SlowestResponseTime, FastestResponseTime,
			memStat.Sys, memStat.Alloc, memStat.TotalAlloc)
		res.Write([]byte(body))
		return
	}

	RequestCount++
	StartTime := time.Now()

	defer func() {
		processDuration := time.Since(StartTime)
		msDuration := uint64(processDuration / time.Millisecond)
		TotalResponseTime += msDuration
		if reflect.ValueOf(SlowestResponseTime).IsZero() {
			SlowestResponseTime = msDuration
		} else {
			if SlowestResponseTime < msDuration {
				SlowestResponseTime = msDuration
			}
		}
		if reflect.ValueOf(FastestResponseTime).IsZero() {
			FastestResponseTime = msDuration
		} else {
			if FastestResponseTime > msDuration {
				FastestResponseTime = msDuration
			}
		}
	}()

	if strings.ToUpper(req.Method) != "GET" {
		recorder := httptest.NewRecorder()
		Execute(15*time.Second, rhh.BackendBaseURL, recorder, req)
		ReturnRecorder(recorder, res)
		return
	}

	breaker := GetBreakerForRequest(req)
	switch breaker.State() {
	case gobreaker.StateOpen:
		ServeFailedProcess(http.StatusBadGateway, res, req, breaker.State())
	default:
		l := serverLog.WithFields(logrus.Fields{
			"Method": req.Method,
		})
		key := getKey(req)
		timeStart := time.Now()
		val, err := breaker.Execute(func() (interface{}, error) {
			l.Debugf("PATH:%s RAWQUERY:%s", req.URL.Path, req.URL.RawQuery)
			recorder := httptest.NewRecorder()
			Execute(15*time.Second, rhh.BackendBaseURL, recorder, req)
			if recorder.Result().StatusCode >= 500 {
				return recorder, fmt.Errorf("response code %d", recorder.Result().StatusCode)
			}
			return recorder, nil
		})
		timeEnd := time.Now()
		recorder := val.(*httptest.ResponseRecorder)
		if err != nil {
			// logrus.Errorf("Error in breaker execution. got %s - code : %d", err.Error(), recorder.Result().StatusCode)
			ServeFailedProcess(recorder.Result().StatusCode, res, req, breaker.State())
		} else {
			recorder.Header().Set("X-Circuit", getGoBreakerString(breaker.State()))
			recorder.Header().Set("X-Retter", "backend")
			ReturnRecorder(recorder, res)
			tx := &DefaultHTTPTransaction{
				TimeStart: timeStart,
				TimeEnd:   timeEnd,
				Rec:       req,
				Res:       recorder,
			}
			cache.Store(key, tx, time.Duration(Config.GetInt(CacheTTL))*time.Second)
			lastKnownSuccess[key] = &DefaultHTTPTransaction{
				TimeStart: timeStart,
				TimeEnd:   timeEnd,
				Rec:       req,
				Res:       recorder,
			}
		}
	}
}

func getGoBreakerString(state gobreaker.State) string {
	switch state {
	case gobreaker.StateOpen:
		return "OPEN"
	case gobreaker.StateHalfOpen:
		return "HALF-OPEN"
	default:
		return "CLOSED"
	}
}

// ServeFailedProcess will be invoked if a call to the Backend server
// failed to be done due to timeout or 5xx errors.
// It will try to look into cache for the cached successful response or
// into history of last known response that was successful
// If no cache or last successful response were found, it will then emit
// 5xx error
func ServeFailedProcess(erroneousResponseCode int, res http.ResponseWriter, req *http.Request, state gobreaker.State) {
	key := getKey(req)
	val := cache.Get(key, false, 0)
	if val == nil {
		if lastSuccessTx, ok := lastKnownSuccess[key]; ok {
			recorder := lastSuccessTx.Response()
			recorder.Header().Set("X-Circuit", getGoBreakerString(state))
			recorder.Header().Set("X-Retter", "last-known-success")
			ReturnRecorder(recorder, res)
			serverLog.Debugf("returned from last success for key %s", key)
		} else {
			res.Header().Set("X-Circuit", getGoBreakerString(state))
			res.Header().Set("X-Retter", "no-cache")
			res.WriteHeader(erroneousResponseCode)
			res.Write([]byte("Backend is down, please try again in few minutes"))
		}
		return
	}
	cachedTx := val.(HTTPTransaction)
	res.Header().Set("X-Circuit", getGoBreakerString(state))
	cachedTx.Response().Header().Set("X-Retter", "cache")
	ReturnRecorder(cachedTx.Response(), res)
}

// ReturnRecorder will write recorded response into response writer
func ReturnRecorder(recorder *httptest.ResponseRecorder, writer http.ResponseWriter) {
	// First we write the headers
	for k, v := range recorder.Header() {
		for _, val := range v {
			writer.Header().Add(k, val)
		}
	}
	// Then we write the status code
	writer.WriteHeader(recorder.Result().StatusCode)
	// Them we write the body if exist
	body, err := ioutil.ReadAll(recorder.Body)
	if err != nil {
		writer.Write([]byte(err.Error()))
		writer.WriteHeader(http.StatusBadGateway)
		return
	}
	writer.Write(body)
}

// Execute will do the actual HTTP call forwarding to the backend server.
// This function is called behind circuit breaker
func Execute(timeout time.Duration, targetURL string, res http.ResponseWriter, req *http.Request) {
	start := time.Now()
	var urlToCall string
	if len(req.URL.RawQuery) > 0 {
		urlToCall = fmt.Sprintf("%s%s?%s", targetURL, req.URL.Path, req.URL.RawQuery)
	} else {
		urlToCall = fmt.Sprintf("%s%s", targetURL, req.URL.Path)
	}
	defer func() {
		duration := time.Since(start)
		serverLog.Tracef("[%s] %s took %d ms", req.Method, urlToCall, duration/time.Millisecond)
	}()
	request, err := http.NewRequest(req.Method, urlToCall, req.Body)
	if err != nil {
		res.Write([]byte(err.Error()))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	request.Header = req.Header
	client := &http.Client{Timeout: timeout}
	response, err := client.Do(request)
	if err != nil {
		if urlErr, yes := err.(*url.Error); yes {
			if urlErr.Timeout() {
				res.Write([]byte(err.Error()))
				res.WriteHeader(RetterStatusBackendTimeout)
				return
			}
		}
		res.Write([]byte(err.Error()))
		res.WriteHeader(http.StatusBadGateway)
		return
	}
	defer response.Body.Close()

	// First we write the headers
	for k, v := range response.Header {
		for _, val := range v {
			res.Header().Add(k, val)
		}
	}
	// Then we write the status code
	res.WriteHeader(response.StatusCode)
	// Them we write the body if exist
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		res.Write([]byte(err.Error()))
		res.WriteHeader(http.StatusBadGateway)
		return
	}
	res.Write(body)
}
