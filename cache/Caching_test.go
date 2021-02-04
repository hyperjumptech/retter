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
