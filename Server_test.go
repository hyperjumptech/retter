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
	"github.com/hyperjumptech/retter/cache"
	"github.com/hyperjumptech/retter/test"
	"go.uber.org/goleak"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRetterHealthCheck(t *testing.T) {
	handler := NewRetterHTTPHandler()
	resp := MakeCall("GET", "/health", t, handler)
	if resp.Result().StatusCode != http.StatusOK {
		t.Fatalf("Health check error")
	}
}

func TestNoCacheNoLastKnown(t *testing.T) {
	defer goleak.VerifyNone(t)

	cache.Clear()

	// lets start our dummy server
	test.StartDummyServer("127.0.0.1:34251", false)
	defer test.StopDummyServer()

	t.Logf("Making dummy server always fail")
	test.FailProbability(1.0)

	Config[BackendURL] = "http://127.0.0.1:34251"
	handler := NewRetterHTTPHandler()

	t.Logf("Making success call")
	resp := MakeCall("GET", "/test/newpath", t, handler)

	if resp.Result().StatusCode != http.StatusInternalServerError || resp.Header().Get("X-Retter") != "no-cache" || resp.Header().Get("X-Circuit") != "CLOSED" {
		t.Fatalf("Unexpected status code %d - retter header %s - circuit status %s", resp.Result().StatusCode, resp.Header().Get("X-Retter"), resp.Header().Get("X-Circuit"))
	}
}

func BenchmarkRetterHTTPHandler_ServeHTTP(b *testing.B) {

	Config[BackendURL] = "http://127.0.0.1:32415"

	handler := NewRetterHTTPHandler()

	for n := 0; n < b.N; n++ {
		r, err := http.NewRequest("GET", "http://localhost/some/path", nil)
		if err != nil {
			b.Fatalf(err.Error())
		}
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, r)
		defer resp.Result().Body.Close()

		if resp.Result().StatusCode != http.StatusOK {
			b.Fatalf("unexpected response %d", resp.Result().StatusCode)
		}
	}
}

func TestRetterHTTPHandler_ServeHTTP(t *testing.T) {
	defer goleak.VerifyNone(t)

	cache.Clear()

	// lets start our dummy server
	test.StartDummyServer("127.0.0.1:34251", false)
	defer test.StopDummyServer()

	Config[BackendURL] = "http://127.0.0.1:34251"
	handler := NewRetterHTTPHandler()

	t.Logf("Making dummy server always success")
	test.FailProbability(0.0)

	t.Logf("Making success call")
	resp := MakeCall("GET", "/test/path", t, handler)
	if resp.Result().StatusCode != 200 || resp.Header().Get("X-Retter") != "backend" || resp.Header().Get("X-Circuit") != "CLOSED" {
		t.Fatalf("Unexpected status code %d - retter header %s", resp.Result().StatusCode, resp.Header().Get("X-Retter"))
	}

	t.Logf("Making dummy server always fail")
	test.FailProbability(1.0)

	for i := 0; i < 3; i++ {
		t.Logf("Making fail #%d while breaker still close", i+1)
		resp := MakeCall("GET", "/test/path", t, handler)
		if resp.Result().StatusCode != http.StatusOK || resp.Header().Get("X-Retter") != "cache" || resp.Header().Get("X-Circuit") != "CLOSED" {
			t.Fatalf("Unexpected status code %d for fail #%d - retter header %s  - circuit %s", resp.Result().StatusCode, i+1, resp.Header().Get("X-Retter"), resp.Header().Get("X-Circuit"))
		}
	}

	// circuit breaker should return with cached success
	t.Logf("Making fail call after circuit open")
	resp = MakeCall("GET", "/test/path", t, handler)
	if resp.Result().StatusCode != http.StatusOK || resp.Header().Get("X-Retter") != "cache" || resp.Header().Get("X-Circuit") != "OPEN" {
		t.Fatalf("Unexpected status code %d - retter header %s  - circuit %s", resp.Result().StatusCode, resp.Header().Get("X-Retter"), resp.Header().Get("X-Circuit"))
	}
}

func MakeCall(method, path string, t *testing.T, handler http.Handler) *httptest.ResponseRecorder {
	r, err := http.NewRequest(method, "http://localhost"+path, nil)
	if err != nil {
		t.Fatalf(err.Error())
		return nil
	}
	r.Header.Add("Accept-Encoding", "gzip")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, r)
	defer resp.Result().Body.Close()

	t.Logf("----------")
	t.Logf("Call result for %s path %s is code %d", method, path, resp.Result().StatusCode)
	for k, v := range resp.Result().Header {
		for _, vv := range v {
			t.Logf("   %s : %s", k, vv)
		}
	}

	return resp
}
