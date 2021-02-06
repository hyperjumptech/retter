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

package cache

import (
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

var (
	log = logrus.WithFields(logrus.Fields{
		"module": "Cache",
		"file":   "cache/Caching.go",
	})
	cacheData = make(map[string]interface{})
	timerData = make(map[string]*time.Timer)
	mutext    sync.Mutex
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
	mutext.Lock()
	defer mutext.Unlock()

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
	mutext.Lock()
	defer mutext.Unlock()

	cacheData[key] = value
	if timer, ok := timerData[key]; ok {
		if !timer.Stop() {
			<-timer.C
		}
		timer.Reset(ttl)
	} else {
		timerData[key] = time.AfterFunc(ttl, func() {
			mutext.Lock()
			defer mutext.Unlock()

			delete(cacheData, key)
			delete(timerData, key)
		})
	}
}

// Get a value from cache identified by the key. It also specify new TTL duration if it need to reset
func Get(key string, reset bool, ttl time.Duration) interface{} {
	mutext.Lock()
	defer mutext.Unlock()

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
	mutext.Lock()
	defer mutext.Unlock()

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
