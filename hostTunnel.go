package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

var tunelCache map[string]string
var mutex = &sync.Mutex{}

func init() {
	tunelCache = make(map[string]string)
	dat, err := ioutil.ReadFile("tunels.config")
	if err != nil {
		fmt.Println(err)
		os.Create("tunels.config")
	} else {
		err = json.Unmarshal(dat, &tunelCache)
		if err != nil {
			fmt.Println(err)
		}
	}
}

// Get the tuneled host for a host
func TunnelHost(ih string) string {
	return tunelCache[ih]
}

func AddTunel(ih string, oh string) {
	mutex.Lock()
	tunelCache[ih] = oh
	mutex.Unlock()
}

func DelTunel(ih string) {
	mutex.Lock()
	delete(tunelCache, ih)
	mutex.Unlock()
}

func GetAllTunels() []string {
	v := make([]string, len(tunelCache))
	idx := 0
	for k, va := range tunelCache {
		v[idx] = k + "." + va
		idx++
	}
	return v
}

func CleanUrl(s string) string {
	s = strings.Replace(s, "https://www.", "", 1)
	s = strings.Replace(s, "http://www.", "", 1)
	return s
}
