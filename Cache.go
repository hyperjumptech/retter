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
	"net/http"
	"net/http/httptest"
	"regexp"
	"time"
)

var (
	cacheLog = logrus.WithFields(logrus.Fields{
		"module": "RetterHTTPHandler",
		"file":   "Server.go",
	})

	// ErrNotFound is an error to be returned if a map do not contain specified key.
	ErrNotFound = fmt.Errorf("RecordNotFound")
	cookieRegex *regexp.Regexp
)

// HTTPTransaction is an interface to store information about HTTP request-response pair,
// the HTTP transaction time and duration.
type HTTPTransaction interface {
	TransactionBeginTime() time.Time
	TransactionDuration() time.Duration
	Request() *http.Request
	Response() *httptest.ResponseRecorder
}

// DefaultHTTPTransaction is the default implementation of HTTPTransaction
type DefaultHTTPTransaction struct {
	TimeStart time.Time
	TimeEnd   time.Time
	Rec       *http.Request
	Res       *httptest.ResponseRecorder
}

// TransactionBeginTime return the time when the transaction begins.
func (tx *DefaultHTTPTransaction) TransactionBeginTime() time.Time {
	return tx.TimeStart
}

// TransactionDuration return the duration of transaction from request till response is captured.
func (tx *DefaultHTTPTransaction) TransactionDuration() time.Duration {
	return tx.TimeEnd.Sub(tx.TimeStart)
}

// Request of the transaction
func (tx *DefaultHTTPTransaction) Request() *http.Request {
	return tx.Rec
}

// Response of the transaction
func (tx *DefaultHTTPTransaction) Response() *httptest.ResponseRecorder {
	return tx.Res
}

func init() {
	regex, err := regexp.Compile(`(ci_session|JSESSIONID|PHPSESSID)\s*=\s*[a-zA-Z0-9.\-]+`)
	if err != nil {
		panic(err)
	}
	cookieRegex = regex
}

func getKey(req *http.Request) string {
	completePath := req.URL.Path
	if Config.GetBoolean(CacheDetectQuery) && len(req.URL.RawQuery) > 0 {
		completePath = fmt.Sprintf("%s?%s", completePath, req.URL.RawQuery)
	}
	if Config.GetBoolean(CacheDetectSession) {
		cookieRow := req.Header.Get("Cookie")
		var cookie string
		if len(cookieRow) > 0 {
			cookie = cookieRegex.FindString(cookieRow)
		}
		if len(cookie) > 0 {
			completePath = fmt.Sprintf("%s:%s", cookie, completePath)
		}
	}
	return completePath
}
