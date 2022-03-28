package app

import (
	"net/http"
	"testing"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/garyburd/redigo/redis"
)

var toxiClient *toxiproxy.Client
var proxies map[string]*toxiproxy.Proxy

func init() {

	var err error

	toxiClient = toxiproxy.NewClient("localhost:8474")
	proxies = make(map[string]*toxiproxy.Proxy)

	// Alternatively, create the proxies manually with
	proxies["redis"], err = toxiClient.CreateProxy("redis", "localhost:26379", "localhost:6379")

	if err != nil {
		panic("cant create proxy")
	}

}

func TestRedisBackendDown(t *testing.T) {
	proxies["redis"].Disable()
	defer proxies["redis"].Enable()

	// Test that redis is down
	_, err := redis.Dial("tcp", ":26379")
	if err == nil {
		t.Fatal("Connection to redis did not fail")
	}
}

func TestRedisBackendSlow(t *testing.T) {
	proxies["redis"].AddToxic("", "latency", "", 1, toxiproxy.Attributes{
		"latency": 1000,
	})
	defer proxies["redis"].RemoveToxic("latency_downstream")

	// Test that redis is slow
	start := time.Now()
	conn, err := redis.Dial("tcp", ":26379")
	if err != nil {
		t.Fatal("Connection to redis failed", err)
	}

	_, err = conn.Do("GET", "test")
	if err != nil {
		t.Fatal("Redis command failed", err)
	} else if time.Since(start) < 900*time.Millisecond {
		t.Fatal("Redis command did not take long enough:", time.Since(start))
	}
}

func TestEphemeralProxy(t *testing.T) {
	proxy, _ := toxiClient.CreateProxy("test", "", "google.com:80")
	defer proxy.Delete()

	// Test connection through proxy.Listen
	resp, err := http.Get("http://" + proxy.Listen)
	if err != nil {
		t.Fatal(err)
	} else if resp.StatusCode != 200 {
		t.Fatal("Proxy to google failed:", resp.StatusCode)
	}
}
