/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package core

import (
	u "github.com/stephencheng/up/utils"
	"testing"
)

type TestObj struct {
	Fa string
	Fb int
	Fc []string
}

func TestCache(t *testing.T) {
	//TODO: fix it
	//u.SetMockConfig()
	u.Pln("start testing")

	c := GetCache()
	c.Put("1key", "key1_value")
	c.Put("2key", "key2_value")
	c.Put("3obj", TestObj{"fa", 24, []string{"a1", "b2"}})

	c.List()

	c.SafeGet("1key")
	v1, ok := c.SafeGet("1key")

	u.Pf("%+v -> %+v\n", ok, v1)
	c.Update("1key", "key1_value_with_update")
	u.Pln(c.Get("1key"))
	c.Obsolete("2key")
	c.List()
	u.Pln("-2key deleted")
	c.Delete("2key")
	c.List()

}

