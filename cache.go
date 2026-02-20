// TTL-based cache (for POST requests per IP)

package main

import (
	"sync"
	"time"
)

var (
	cache  map[string]time.Time
	locker *sync.RWMutex
)

func init() {
	cache = make(map[string]time.Time)
	locker = &sync.RWMutex{}

	go func() {
		for {
			cleanCache()
			time.Sleep(ipDelaySec / 2 * time.Second)
		}
	}()
}

func cleanCache() {
	locker.Lock()
	defer locker.Unlock()
	for ip, expire := range cache {
		if time.Now().After(expire) {
			delete(cache, ip)
			//log.Println("Deleted from IP cache:", ip)
		}
	}
}

func setCache(ip string) {
	var expire time.Time

	expire = time.Now().Add(ipDelaySec * time.Second)
	locker.Lock()
	cache[ip] = expire
	locker.Unlock()
}

func isCached(ip string) bool {
	locker.Lock()
	defer locker.Unlock()
	if _, ok := cache[ip]; ok {
		return true
	}
	return false
}
