package main

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type requestContext struct {
	cache *RequestCache
	host  string
	wr    *http.ResponseWriter
	rq    *http.Request
	body  []byte
	hash  [32]byte
}

var port = ":80"

func doExternalRequest(r *http.Request, host string) *http.Response {
	request, err := http.NewRequest(r.Method, host+r.RequestURI, r.Body)

	cleanuri := CleanUrl(host)
	for _, cookie := range r.Cookies() {
		ncookie := http.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Expires:  cookie.Expires,
			Domain:   cleanuri,
			Secure:   false,
			HttpOnly: false,
		}
		request.AddCookie(&ncookie)
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return resp
}

func tunnelRequest(w *http.ResponseWriter, r *http.Request, cache *RequestCache, host string) bool {

	resp := doExternalRequest(r, host)
	if resp == nil {
		return false
	}
	defer resp.Body.Close()

	bodyRead, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return false
	}

	cache.cookies = make([]http.Cookie, len(resp.Cookies()))
	cache.body = bodyRead
	cache.headers = resp.Header
	cache.expiration = time.Now().Add(time.Minute * 5)

	for v, cookie := range resp.Cookies() {
		ncookie := http.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Expires:  cookie.Expires,
			Domain:   strings.Replace(r.Host, port, "", 1),
			Secure:   false,
			HttpOnly: false,
		}
		cache.cookies[v] = ncookie
	}

	delete(cache.headers, "Set-Cookie")
	for header, value := range resp.Header {
		for i := 0; i < len(value); i++ {
			(*w).Header().Set(header, value[i])
		}
	}

	(*w).Write(bodyRead)
	return true
}

func pipeline(w http.ResponseWriter, r *http.Request) {
	var context = &requestContext{wr: &w, rq: r, cache: &RequestCache{}}
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		context.body = content
	}
	context.host = TunnelHost(strings.Replace(r.Host, port, "", 1))
	context.hash = createHash(r, &context.body)

	addCorsHeaders(context.wr, context.rq)
	CacheRequest(context, func(context *requestContext) bool {
		return tunnelRequest(context.wr, context.rq, context.cache, context.host)
	})
}

func addCorsHeaders(w *http.ResponseWriter, r *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Credentials", "true")
	(*w).Header().Set("Access-Control-Allow-Headers", "authorization,content-type,x-apiinternalkey")
	(*w).Header().Set("Access-Control-Allow-Methods", r.Method)
	(*w).Header().Set("Access-Control-Allow-Origin", r.Host)
}

func createHash(r *http.Request, body *[]byte) [32]byte {
	return sha256.Sum256(append([]byte(r.Host+r.RequestURI+r.Method), *body...))
}

func main() {
	http.HandleFunc("/", pipeline)
	if err := http.ListenAndServe(port, nil); err != nil {
		panic(err)
	}
}
