package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"time"
)

var (
	ErrNotFound = fmt.Errorf("RecordNotFound")
	cookieRegex *regexp.Regexp
)

type Transaction struct {
	TimeEntry time.Time
	TimeStart time.Time
	TimeEnd   time.Time
	Rec       *http.Request
	Res       *httptest.ResponseRecorder
}

func init() {
	regex, err := regexp.Compile(`(ci_session|JSESSIONID|PHPSESSID)\s*=\s*[a-zA-Z0-9.\-]+`)
	if err != nil {
		panic(err)
	}
	cookieRegex = regex
}

func NewTransactionRecord() TransactionRecord {
	return make(TransactionRecord)
}

type TransactionRecord map[string]*Transaction

func (tr TransactionRecord) Store(tx *Transaction) {
	completePath := tx.Rec.URL.Path
	if len(tx.Rec.URL.RawQuery) > 0 {
		completePath = fmt.Sprintf("%s?%s", completePath, tx.Rec.URL.RawQuery)
	}
	cookieRow := tx.Rec.Header.Get("Cookie")
	var cookie string
	if len(cookieRow) > 0 {
		cookie = cookieRegex.FindString(cookieRow)
	}
	tx.TimeEntry = time.Now()
	// store the generic result
	tr[completePath] = tx
	// store the result with cookie
	tr[fmt.Sprintf("%s:%s", cookie, completePath)] = tx
}

func (tr TransactionRecord) GetCached(req *http.Request) (*Transaction, error) {
	completePath := req.URL.Path
	if len(req.URL.RawQuery) > 0 {
		completePath = fmt.Sprintf("%s?%s", completePath, req.URL.RawQuery)
	}
	cookieRow := req.Header.Get("Cookie")
	var cookie string
	if len(cookieRow) > 0 {
		cookie = cookieRegex.FindString(cookieRow)
	}
	if tx, ok := tr[fmt.Sprintf("%s:%s", cookie, completePath)]; ok {
		return tx, nil
	}
	if tx, ok := tr[completePath]; ok {
		return tx, nil
	}
	return nil, ErrNotFound
}
