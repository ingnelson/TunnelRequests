package main

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/fatih/color"
	"github.com/nemeq/ServerTunnel/syncsp"
)

var cache map[[32]byte]RequestCache
var spinLock = syncsp.SpinLock{}

type RequestCache struct {
	headers    http.Header
	body       bytes.Buffer
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
		go color.White("302 from cache " + context.host + context.rq.RequestURI)
		tunnelCacheResponse(cache, context.wr)
		return
	} else {
		go color.Red("Not found in cache and request error.")
		(*context.wr).WriteHeader(404)
		io.Copy((*context.wr), &context.cache.body)
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
	(*w).Header().Set("CustomCache", "true")
	io.Copy(*w, &cache.body)
	return
}
