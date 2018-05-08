package main

import (
	"net/http"
	"time"

	"github.com/nemeq/ServerTunnel/sync"
)

var cache map[[32]byte]RequestCache
var spinLock = sync.SpinLock{}

type RequestCache struct {
	headers    http.Header
	body       []byte
	cookies    []http.Cookie
	expiration time.Time
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

func CacheRequest(context *requestContext, tunnelContinue func(context *requestContext) bool) {
	if tunnelContinue(context) {
		go AddRequestToCache(context.hash, *context.cache)
		return
	}

	cache := GetCachedRequest(context.hash)
	if cache != nil {
		tunnelCacheResponse(cache, context.wr)
	} else {
		http.NotFound(*context.wr, context.rq)
	}

}

func tunnelCacheResponse(cache *RequestCache, w *http.ResponseWriter) {
	for _, cookie := range cache.cookies {
		http.SetCookie(*w, &cookie)
	}

	for header, value := range cache.headers {
		for i := 0; i < len(value); i++ {
			(*w).Header().Set(header, value[i])
		}
	}

	(*w).Write(cache.body)
	return
}
