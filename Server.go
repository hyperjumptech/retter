package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"
)

var (
	serverLog = logrus.WithFields(logrus.Fields{
		"module": "RetterHttpHandler",
		"file":   "Server.go",
	})
)

func NewRetterHttpHandler(BackendBaseURL string) http.Handler {
	return &RetterHttpHandler{
		BackendBaseURL: BackendBaseURL,
		TxRecord:       NewTransactionRecord(),
	}
}

type RetterHttpHandler struct {
	BackendBaseURL string
	TxRecord       TransactionRecord
}

func (rhh *RetterHttpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	l := serverLog.WithFields(logrus.Fields{
		"Method": req.Method,
	})
	l.Debugf("PATH:%s RAWQUERY:%s", req.URL.Path, req.URL.RawQuery)
	recorder := httptest.NewRecorder()
	rhh.Execute(15*time.Second, rhh.BackendBaseURL, recorder, req)
	rhh.ReturnRecorder(recorder, res)
}

func (rhh *RetterHttpHandler) ReturnRecorder(recorder *httptest.ResponseRecorder, writer http.ResponseWriter) {
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

func (rhh *RetterHttpHandler) Execute(timeout time.Duration, targetURL string, res http.ResponseWriter, req *http.Request) {
	start := time.Now()
	var urlToCall string
	if len(req.URL.RawQuery) > 0 {
		urlToCall = fmt.Sprintf("%s%s?%s", targetURL, req.URL.Path, req.URL.RawQuery)
	} else {
		urlToCall = fmt.Sprintf("%s%s", targetURL, req.URL.Path)
	}
	defer func() {
		duration := time.Since(start)
		serverLog.Infof("[%s] %s took %d ms", req.Method, urlToCall, duration/time.Millisecond)
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
