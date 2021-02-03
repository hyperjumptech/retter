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

	cache            = make(map[string]HTTPTransaction)
	lastKnownSuccess = make(map[string]HTTPTransaction)
	ttlTimer         = make(map[string]*AbortableDeadlineTimer)

	txRemoveChannel  = make(chan string)
	txStoreChannel   = make(chan HTTPTransaction)
	stopCacheChannel = make(chan bool)
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

// CacheStop will stop the cache. Call this when server is shut-down
func CacheStop() {
	stopCacheChannel <- true
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
	go cacheSelect()
}

// AbortableDeadlineTimer a structure that hold a timer, and a channel to abort the channel
type AbortableDeadlineTimer struct {
	deadlineTimer *time.Timer
	abort         chan bool
}

func cacheSelect() {
	for {
		select {
		case <-stopCacheChannel:
			for _, timer := range ttlTimer {
				timer.abort <- true
			}
			return
		case str := <-txRemoveChannel:
			if abortable, ok := ttlTimer[str]; ok {
				abortable.deadlineTimer.Stop()
				delete(ttlTimer, str)
			}
			delete(cache, str)
		case tx := <-txStoreChannel:
			key := getKey(tx.Request())

			// if deadlineTimer for key to override exist, lets stop the deadlineTimer and remove
			// the deadlineTimer for our ttlTable.
			if abortable, ok := ttlTimer[key]; ok {
				abortable.deadlineTimer.Stop()
				abortable.abort <- true
				delete(ttlTimer, key)
			}
			// if cache for key to replace exist, lets remove the cache first.
			if _, ok := cache[key]; ok {
				delete(cache, key)
			}

			// add the deadlineTimer into our map.
			cache[key] = tx

			// create an abortable deadlineTimer for our cache TTL
			abortable := &AbortableDeadlineTimer{
				deadlineTimer: time.NewTimer(time.Duration(Config.GetInt(CacheTTL)) * time.Second),
				abort:         make(chan bool),
			}
			ttlTimer[key] = abortable

			// create a go routine to scan for our deadlineTimer
			go func() {
				select {
				case <-abortable.deadlineTimer.C:
					txRemoveChannel <- key
				case <-abortable.abort:
					// its abborted
				}
			}()
		}
	}
}

// CacheStore stores a transaction into cache. The cache will have TTL.
// Cache entry with same key will replace the old one.
func CacheStore(txStart, txEnd time.Time, req *http.Request, res *httptest.ResponseRecorder) {
	txStoreChannel <- &DefaultHTTPTransaction{
		TimeStart: txStart,
		TimeEnd:   txEnd,
		Rec:       req,
		Res:       res,
	}
}

// CacheGet will return a transaction with same generated cache key, if the entry is not yet expired.
func CacheGet(req *http.Request, resetTTL bool) (HTTPTransaction, error) {
	key := getKey(req)
	if tx, ok := cache[key]; ok {
		if resetTTL {
			if tmr, ok := ttlTimer[key]; ok {
				tmr.deadlineTimer.Reset(time.Duration(Config.GetInt(CacheTTL)) * time.Second)
			} else {
				cacheLog.Fatalf("Cache key %s have no corresponding TTL Timer", key)
			}
		}
		return tx, nil
	}
	return nil, ErrNotFound
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
