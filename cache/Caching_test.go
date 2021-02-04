package cache

import (
	"go.uber.org/goleak"
	"testing"
	"time"
)

func TestCacheNoReset(t *testing.T) {
	defer goleak.VerifyNone(t)

	Store("akey", "avalue", 2*time.Second)
	val := Get("akey", false, 0)
	if val.(string) != "avalue" {
		t.Errorf("Expect \"avalue\" but \"%s\"", val.(string))
	}
	time.Sleep(1 * time.Second)
	val = Get("akey", false, 0)
	if val.(string) != "avalue" {
		t.Errorf("Expect \"avalue\" but \"%s\"", val.(string))
	}
	time.Sleep(2 * time.Second)
	val = Get("akey", false, 0)
	if val != nil {
		t.Errorf("Expect nil but \"%s\"", val.(string))
	}
}

func TestCacheWithReset(t *testing.T) {
	defer goleak.VerifyNone(t)

	Store("akey", "avalue", 1*time.Second)
	val := Get("akey", false, 0)
	if val.(string) != "avalue" {
		t.Errorf("Expect \"avalue\" but \"%s\"", val.(string))
	}
	time.Sleep(500 * time.Millisecond)
	val = Get("akey", true, 1*time.Second)
	if val.(string) != "avalue" {
		t.Errorf("Expect \"avalue\" but \"%s\"", val.(string))
	}
	time.Sleep(500 * time.Millisecond)
	val = Get("akey", true, 1*time.Second)
	if val.(string) != "avalue" {
		t.Errorf("Expect \"avalue\" but \"%s\"", val.(string))
	}
	time.Sleep(500 * time.Millisecond)
	val = Get("akey", false, 0)
	if val.(string) != "avalue" {
		t.Errorf("Expect \"avalue\" but \"%s\"", val.(string))
	}
	time.Sleep(600 * time.Millisecond)
	val = Get("akey", false, 0)
	if val != nil {
		t.Errorf("Expect nil but \"%s\"", val.(string))
	}
}
