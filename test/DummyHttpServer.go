package test

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

var (
	DummyServer      *http.Server
	DummyServerAlive = false
	dummyHttpHandler = &DummyHttpHandler{
		fastest:         0,
		slowest:         1 * time.Second,
		failProbability: 0,
	}
	RequestCount = 0
)

func SetFastest(f, s time.Duration) {
	dummyHttpHandler.fastest = f
	dummyHttpHandler.slowest = s
	if dummyHttpHandler.fastest > dummyHttpHandler.slowest {
		t := dummyHttpHandler.fastest
		dummyHttpHandler.fastest = dummyHttpHandler.slowest
		dummyHttpHandler.slowest = t
	}
}

func FailProbability(f float64) {
	dummyHttpHandler.failProbability = f
}

func StartDummyServer(addr string, standalone bool) {
	if DummyServerAlive {
		return
	}

	DummyServer = &http.Server{
		Addr: addr,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: 1 * time.Minute,
		ReadTimeout:  1 * time.Minute,
		IdleTimeout:  1 * time.Minute,
		Handler:      dummyHttpHandler, // Pass our instance of gorilla/mux in.
	}

	if standalone {
		fmt.Printf("Dummyserver is listening on : %s\n", DummyServer.Addr)
		if err := DummyServer.ListenAndServe(); err != nil {
			log.Println(err)
		}
	} else {
		go func() {
			fmt.Printf("Dummyserver is listening on : %s\n", DummyServer.Addr)
			DummyServerAlive = true
			if err := DummyServer.ListenAndServe(); err != nil {
				log.Println(err)
			}
			DummyServerAlive = false
		}()
	}
}

func StopDummyServer() {
	DummyServer.Shutdown(context.Background())
}

type DummyHttpHandler struct {
	fastest         time.Duration
	slowest         time.Duration
	failProbability float64
}

func (dhh *DummyHttpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	RequestCount++
	rc := RequestCount
	if req.URL.Path == "/set" {
		newFast := req.URL.Query().Get("f")
		newSlow := req.URL.Query().Get("s")
		newError := req.URL.Query().Get("e")
		if len(newFast) > 0 {
			nf, err := strconv.Atoi(newFast)
			if err == nil {
				dhh.fastest = time.Duration(nf) * time.Second
			}
		}
		if len(newSlow) > 0 {
			ns, err := strconv.Atoi(newSlow)
			if err == nil {
				dhh.slowest = time.Duration(ns) * time.Second
			}
		}
		if len(newError) > 0 {
			ne, err := strconv.ParseFloat(newError, 64)
			if err == nil {
				dhh.failProbability = ne
			}
		}
		if dhh.fastest > dhh.slowest {
			t := dhh.fastest
			dhh.fastest = dhh.slowest
			dhh.slowest = t
		}
		res.Header().Set("Content-Type", "text/plain")
		res.WriteHeader(http.StatusOK)
		res.Write([]byte(fmt.Sprintf("DONE %d", rc)))
	} else {
		dur := dhh.slowest - dhh.fastest
		sleep := dhh.fastest + time.Duration(rand.Int63n(int64(dur)))
		time.Sleep(sleep)
		res.Header().Set("Content-Type", "text/plain")
		failRandom := rand.Float64()
		if failRandom < dhh.failProbability {
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte(fmt.Sprintf("ERROR %d", rc)))
		} else {
			res.WriteHeader(http.StatusOK)
			res.Write([]byte(fmt.Sprintf("DONE %d", rc)))
		}
	}
}
