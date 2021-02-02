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
		"module": "RetterHttpHandler",
		"file":   "Server.go",
	})

	ErrNotFound = fmt.Errorf("RecordNotFound")
	cookieRegex *regexp.Regexp
	Ttl         = 1 * time.Minute

	cache            = make(map[string]HTTPTransaction)
	lastKnownSuccess = make(map[string]HTTPTransaction)
	ttlTimer         = make(map[string]*AbortableDeadlineTimer)

	txRemoveChannel  = make(chan string)
	txStoreChannel   = make(chan HTTPTransaction)
	stopCacheChannel = make(chan bool)
)

type HTTPTransaction interface {
	TransactionBeginTime() time.Time
	TransactionDuration() time.Duration
	Request() *http.Request
	Response() *httptest.ResponseRecorder
}

type DefaultHTTPTransaction struct {
	TimeStart time.Time
	TimeEnd   time.Time
	Rec       *http.Request
	Res       *httptest.ResponseRecorder
}

func CacheStop() {
	stopCacheChannel <- true
}

func (tx *DefaultHTTPTransaction) TransactionBeginTime() time.Time {
	return tx.TimeStart
}
func (tx *DefaultHTTPTransaction) TransactionDuration() time.Duration {
	return tx.TimeEnd.Sub(tx.TimeStart)
}
func (tx *DefaultHTTPTransaction) Request() *http.Request {
	return tx.Rec
}
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
				deadlineTimer: time.NewTimer(time.Duration(Config.GetInt(CACHE_TTL)) * time.Second),
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

func CacheStore(txStart, txEnd time.Time, req *http.Request, res *httptest.ResponseRecorder) {
	txStoreChannel <- &DefaultHTTPTransaction{
		TimeStart: txStart,
		TimeEnd:   txEnd,
		Rec:       req,
		Res:       res,
	}
}

func CacheGet(req *http.Request, resetTtl bool) (HTTPTransaction, error) {
	key := getKey(req)
	if tx, ok := cache[key]; ok {
		if resetTtl {
			if tmr, ok := ttlTimer[key]; ok {
				tmr.deadlineTimer.Reset(time.Duration(Config.GetInt(CACHE_TTL)) * time.Second)
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
	if len(req.URL.RawQuery) > 0 {
		completePath = fmt.Sprintf("%s?%s", completePath, req.URL.RawQuery)
	}
	if Config.GetBoolean(CACHE_DETECTSESSION) {
		cookieRow := req.Header.Get("Cookie")
		var cookie string
		if len(cookieRow) > 0 {
			cookie = cookieRegex.FindString(cookieRow)
		}
		return fmt.Sprintf("%s:%s", cookie, completePath)
	}
	return completePath
}
