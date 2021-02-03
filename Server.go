package main

import (
	"fmt"
	"github.com/hyperjumptech/jiffy"
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
		cacheCount := len(cache)
		timerCount := len(ttlTimer)
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
		ServeFailedProcess(res, req)
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
			l.Debugf("Request key %s error with response code %d", key, recorder.Result().StatusCode)
			ServeFailedProcess(res, req)
		} else {
			ReturnRecorder(recorder, res)
			CacheStore(timeStart, timeEnd, req, recorder)
			lastKnownSuccess[key] = &DefaultHTTPTransaction{
				TimeStart: timeStart,
				TimeEnd:   timeEnd,
				Rec:       req,
				Res:       recorder,
			}
		}
	}
}

// ServeFailedProcess will be invoked if a call to the Backend server
// failed to be done due to timeout or 5xx errors.
// It will try to look into cache for the cached successful response or
// into history of last known response that was successful
// If no cache or last successful response were found, it will then emit
// 5xx error
func ServeFailedProcess(res http.ResponseWriter, req *http.Request) {
	cachedTx, err := CacheGet(req, true)
	if err != nil {
		if err == ErrNotFound {
			key := getKey(req)
			if lastSuccessTx, ok := lastKnownSuccess[key]; ok {
				ReturnRecorder(lastSuccessTx.Response(), res)
				serverLog.Debugf("returned from last success for key %s", key)
			} else {
				res.Write([]byte("Backend is down, please try again in few minutes"))
				res.WriteHeader(http.StatusBadGateway)
			}
			return
		}
		serverLog.Errorf("error while fetch request for cache in circuit open")
		res.Write([]byte(fmt.Sprintf("Retter cache error. got %s", err.Error())))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
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
