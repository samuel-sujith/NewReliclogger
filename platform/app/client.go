package app

import (
	"fmt"
	"net/http"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/garyburd/redigo/redis"
)

func (S *Stub) RedisBackendDown(w http.ResponseWriter, r *http.Request) {
	// Test that redis is down
	S.RedisProxy.Disable()
	defer S.RedisProxy.Enable()
	_, err := redis.Dial("tcp", ":7379")
	if err != nil {
		S.RedisDownHits.Increment()
		fmt.Println("increm red down")
	}
}

func (S *Stub) RedisBackendUp(w http.ResponseWriter, r *http.Request) {
	// Test that redis is Up
	_, err := redis.Dial("tcp", ":7379")
	if err == nil {
		S.RedisUpHits.Increment()
		fmt.Println("increm red up")
	}
}

func (S *Stub) GitcheckUp(w http.ResponseWriter, r *http.Request) {

	// Test connection through proxy.Listen
	fmt.Println("gitup proxy listen", S.ToxiProxy.Listen)
	resp, err := http.Get("http://" + S.ToxiProxy.Listen)
	fmt.Println("response from gitup", resp)
	if err == nil {
		S.GitDownHits.Increment()
	} else {
		S.GitDownHits.Increment()
	}
}

func (S *Stub) GitcheckDown(w http.ResponseWriter, r *http.Request) {

	fmt.Println("Git checkdown proxy is ", S.ToxiProxy)
	S.ToxiProxy.Disable() //Disabling the proxy will simulate git down
	defer S.ToxiProxy.Enable()

	// Test connection through proxy.Listen
	resp, err := http.Get("http://" + S.ToxiProxy.Listen)
	fmt.Println("response from gitdown", resp)

	if err != nil {
		S.GitDownHits.Increment()
	} else if resp.StatusCode != 200 {
		S.GitDownHits.Increment()
	}
}

func (S *Stub) RedisBackendSlow(w http.ResponseWriter, r *http.Request) {

	S.RedisProxy.AddToxic("latency_downstream", "latency", "", 1, toxiproxy.Attributes{
		"latency": 1000,
	}) //This adds latency to the Redis connect request
	defer S.RedisProxy.RemoveToxic("latency_downstream")

	// Test that redis is slow
	start := time.Now()
	_, err := redis.Dial("tcp", ":7379")

	if err != nil {
		S.RedisDownHits.Increment()
	} else if time.Since(start) < 900*time.Millisecond {
		S.RedisUpHits.Increment()
	} else {
		S.RedisSlowHits.Increment()
	}
}
