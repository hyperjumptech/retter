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
	"fmt"
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

func TestWriteReadRemove(t *testing.T) {
	defer goleak.VerifyNone(t)

	Clear()

	if CacheSize() != 0 {
		t.Errorf("Excpect cache size = 0 but %d", CacheSize())
	}
	if TimerSize() != 0 {
		t.Errorf("Excpect timer size = 0 but %d", TimerSize())
	}

	tmr := time.Now()
	for i := 0; i < 5000; i++ {
		k := fmt.Sprintf("K%d", i)
		v := fmt.Sprintf("V%d", i)
		Store(k, v, 2*time.Second)
		vget := Get(k, false, 0)
		if vget.(string) != v {
			t.Errorf("expect equals %s, but %s", v, vget)
		}
		if i >= 3000 {
			Remove(k)
		}
	}
	if (time.Since(tmr) / time.Millisecond) >= (2000 * time.Millisecond) {
		t.Fatalf("not enought ime to store get and remove")
	}

	time.Sleep(200 * time.Millisecond)

	if CacheSize() != TimerSize() {
		t.Fatalf("Cache %d != Timer %d", CacheSize(), TimerSize())
	}
	if CacheSize() != 3000 {
		t.Fatalf("Excpect cache size = 3000 but %d", CacheSize())
	}
	if TimerSize() != 3000 {
		t.Fatalf("Excpect timer size = 3000 but %d", TimerSize())
	}

	time.Sleep(2 * time.Second)

	if CacheSize() != TimerSize() {
		t.Fatalf("Cache %d != Timer %d", CacheSize(), TimerSize())
	}
	if CacheSize() != 0 {
		t.Fatalf("Excpect cache size = 3000 but %d", CacheSize())
	}
	if TimerSize() != 0 {
		t.Fatalf("Excpect timer size = 3000 but %d", TimerSize())
	}
}

func BenchmarkCache(b *testing.B) {
	for i := 0; i < b.N; i++ {
		k := fmt.Sprintf("K%d", i)
		v := fmt.Sprintf("V%d", i)
		Store(k, v, 2*time.Second)
		vget := Get(k, false, 0)
		if vget.(string) != v {
			b.Errorf("expect equals %s, but %s", v, vget)
		}
		if i >= 3000 {
			Remove(k)
		}
	}
}

func TestCacheWithReset(t *testing.T) {
	defer goleak.VerifyNone(t)

	Clear()

	if CacheSize() != 0 {
		t.Errorf("Excpect cache size = 0 but %d", CacheSize())
	}
	if TimerSize() != 0 {
		t.Errorf("Excpect timer size = 0 but %d", TimerSize())
	}

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
	if CacheSize() != 1 {
		t.Errorf("Excpect cache size = 1 but %d", CacheSize())
	}
	if TimerSize() != 1 {
		t.Errorf("Excpect timer size = 1 but %d", TimerSize())
	}
	time.Sleep(500 * time.Millisecond)
	val = Get("akey", false, 0)
	if val.(string) != "avalue" {
		t.Errorf("Expect \"avalue\" but \"%s\"", val.(string))
	}
	time.Sleep(600 * time.Millisecond)
	if CacheSize() != 0 {
		t.Errorf("Excpect cache size = 0 but %d", CacheSize())
	}
	if TimerSize() != 0 {
		t.Errorf("Excpect timer size = 0 but %d", TimerSize())
	}
	val = Get("akey", false, 0)
	if val != nil {
		t.Errorf("Expect nil but \"%s\"", val.(string))
	}
}
