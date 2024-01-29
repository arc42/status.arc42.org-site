package main

import (
	"fmt"
	"time"
	"zgo.at/zcache/v2"

	// go-cache is not actively maintained any longer
	"github.com/patrickmn/go-cache"
)

// small test of the caching package "/go-cache v2.1.0+incompatible"
// source inspired by https://github.com/patrickmn/go-cache

func main() {
	// Create a cache with a default expiration time of 5 minutes, and which
	// purges expired items every 10 minutes
	c := cache.New(5*time.Minute, 10*time.Minute)

	// in zcache we need to use the function `SetWithExpire`
	z := zcache.New[string, any](zcache.NoExpiration, zcache.NoExpiration)
	z.SetWithExpire("foo", "bar", zcache.DefaultExpiration)

	// Set the value of the key "foo" to "bar", with the default expiration time
	c.Set("foo", "bar", cache.DefaultExpiration)

	// set "exp" to expire after 5 Milliseconds
	c.Set("exp", "42", 3000*time.Millisecond)

	c.Set("baz", 42, cache.NoExpiration)

	// wait for 1 second
	time.Sleep(time.Second)

	// Get the string associated with the key "foo" from the cache
	foo, found := c.Get("foo")
	if found {
		fmt.Println(foo)
	}

	exp, found := c.Get("exp")
	if found {
		fmt.Println(exp)
	} else {
		fmt.Println("exp not found in cache, expired?")
	}
}
