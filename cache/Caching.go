package cache

import (
	"github.com/sirupsen/logrus"
	"time"
)

var (
	log = logrus.WithFields(logrus.Fields{
		"module": "Cache",
		"file":   "cache/Caching.go",
	})
	cacheData = make(map[string]interface{})
	timerData = make(map[string]*time.Timer)
)

// CacheSize return the size of this cache
func CacheSize() int {
	return len(cacheData)
}

// TimerSize return the size of timer
func TimerSize() int {
	return len(timerData)
}

// Clear the cache
func Clear() {
	dataKeys := make([]string, len(cacheData))
	index := 0
	for k, _ := range cacheData {
		dataKeys[index] = k
		index++
	}
	for _, v := range dataKeys {
		delete(cacheData, v)
	}

	timerKeys := make([]string, len(timerData))
	index = 0
	for k, _ := range timerData {
		timerKeys[index] = k
		index++
	}
	for _, v := range timerKeys {
		delete(timerData, v)
	}
}

// Store a value into cache identified by the key. It also specify the TTL duration
func Store(key string, value interface{}, ttl time.Duration) {
	cacheData[key] = value
	if timer, ok := timerData[key]; ok {
		if !timer.Stop() {
			<-timer.C
		}
		timer.Reset(ttl)
	} else {
		timerData[key] = time.AfterFunc(ttl, func() {
			delete(cacheData, key)
			delete(timerData, key)
		})
	}
}

// Get a value from cache identified by the key. It also specify new TTL duration if it need to reset
func Get(key string, reset bool, ttl time.Duration) interface{} {
	if value, ok := cacheData[key]; ok {
		if timer, ok := timerData[key]; ok && reset {
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(ttl)
		}
		return value
	}
	return nil
}

// Remove a cache entry
func Remove(key string) {
	if timer, ok := timerData[key]; ok {
		if !timer.Stop() {
			<-timer.C
		}
		delete(timerData, key)
	}
	if _, ok := cacheData[key]; ok {
		delete(cacheData, key)
	}
}
