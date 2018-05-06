package main

import (
	"net/http"

	"github.com/nemeq/ServerTunnel/sync"
)

var cache map[[32]byte]RequestCache
var spinLock = sync.SpinLock{}

type RequestCache struct {
	headers http.Header
	body    []byte
	cookies []http.Cookie
}

func init() {
	cache = make(map[[32]byte]RequestCache)
}

func GetCachedRequest(ih [32]byte) *RequestCache {
	cache, found := cache[ih]
	if found {
		return &cache
	}
	return nil
}

func AddRequestToCache(ih [32]byte, oh RequestCache) {
	spinLock.Lock()
	cache[ih] = oh
	spinLock.Unlock()
}

func DeleteRequest(ih [32]byte) {
	spinLock.Lock()
	delete(cache, ih)
	spinLock.Unlock()
}

func CacheRequest(context *requestContext, tunnelContinue func(context *requestContext)) {
	cache := GetCachedRequest(context.hash)
	if cache != nil {
		for _, cookie := range cache.cookies {
			http.SetCookie(*context.wr, &cookie)
		}

		for header, value := range cache.headers {
			for i := 0; i < len(value); i++ {
				(*context.wr).Header().Set(header, value[i])
			}
		}

		(*context.wr).Write(cache.body)
		return
	}
	tunnelContinue(context)
	go AddRequestToCache(context.hash, *context.cache)
}
