package app

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
)

//Main Stub for handling the endpoint requests
type Stub struct {
	h              *telemetry.Harvester
	databaseHits   *telemetry.AggregatedCount
	databaseMisses *telemetry.AggregatedCount
	RedisUpHits    *telemetry.AggregatedCount
	RedisDownHits  *telemetry.AggregatedCount
	RedisSlowHits  *telemetry.AggregatedCount
	GitDownHits    *telemetry.AggregatedCount
	GitUpHits      *telemetry.AggregatedCount
	ToxiProxy      *toxiproxy.Proxy
	RedisProxy     *toxiproxy.Proxy
	ToxiClient     *toxiproxy.Client
}

//legacy telemetry code
func (S *Stub) databaseCall(collection string) {
	if rand.Intn(10) < 5 {
		S.databaseMisses.Increment()
		time.Sleep(10)
	} else {
		S.databaseHits.Increment()
		time.Sleep(1)
	}
}

//legacy telemetry code
func (S *Stub) fetch(w http.ResponseWriter, r *http.Request) {
	S.databaseCall("users")
	io.WriteString(w, "fetch!")
}

//legacy telemetry code
func (S *Stub) index(w http.ResponseWriter, r *http.Request) {
	time.Sleep(5)
	io.WriteString(w, "index!")
}

//legacy telemetry code
func (S *Stub) outboundCall(u *url.URL) {
	req, _ := http.NewRequest("GET", u.String(), nil)

	before := time.Now()
	http.DefaultClient.Do(req)

	statuses := []int{200, 200, 200, 200, 200, 404, 503}
	status := statuses[rand.Int()%len(statuses)]

	S.h.MetricAggregator().Summary("service.span.responseTime", map[string]interface{}{
		"host":        u.Host,
		"method":      "GET",
		"http.status": status,
	}).RecordDuration(time.Since(before))
}

//legacy telemetry code
func (S *Stub) outbound(w http.ResponseWriter, r *http.Request) {
	u, _ := url.Parse("http://www.example.com")
	S.outboundCall(u)
	io.WriteString(w, "outbound!")
}

func randomID() string {
	// rand.Uint64 is Go 1.8+
	u1 := rand.Uint32()
	u2 := rand.Uint32()
	u := (uint64(u1) << 32) | uint64(u2)
	return fmt.Sprintf("%016x", u)
}

//Generates span for each request which is sent to :8000 server
func (S *Stub) wrapHandler(path string, exceptype string, handler func(http.ResponseWriter, *http.Request)) (string, func(http.ResponseWriter, *http.Request)) {
	return path, func(rw http.ResponseWriter, req *http.Request) {
		s := S.h.MetricAggregator().Summary("service.responseTime", map[string]interface{}{
			"name":                path,
			"http.method":         req.Method,
			"isWeb":               true,
			"service.instance.id": "New relic trial metric",
		})
		before := time.Now()
		handler(rw, req)
		s.RecordDuration(time.Since(before))

		S.h.RecordSpan(telemetry.Span{
			ID:          randomID(),
			TraceID:     randomID(),
			Name:        "service.responseTime",
			Timestamp:   before,
			Duration:    time.Since(before),
			ServiceName: "NR assignment exp 1 Stub",
			Attributes: map[string]interface{}{
				"name":                path,
				"http.method":         req.Method,
				"isWeb":               true,
				"service.instance.id": "New relic trial span",
			},
			Events: []telemetry.Event{
				telemetry.Event{
					EventType: "exception",
					Timestamp: before,
					Attributes: map[string]interface{}{
						"exception.message":   "Checking",
						"exception.type":      exceptype,
						"service.instance.id": "New relic trial event",
					},
				},
			},
		})

		S.h.RecordEvent(telemetry.Event{
			EventType: "CustomEvent",
			Timestamp: before,
			Attributes: map[string]interface{}{
				"path":                path,
				"http.method":         req.Method,
				"isWeb":               true,
				"service.instance.id": "New relic trial event",
			},
		})
	}
}

func (S *Stub) gatherMemStats() {
	allocations := S.h.MetricAggregator().Gauge("runtime.MemStats.heapAlloc", map[string]interface{}{})
	var rtm runtime.MemStats
	var interval = 1 * time.Second
	for {
		<-time.After(interval)
		runtime.ReadMemStats(&rtm)
		allocations.Value(float64(rtm.HeapAlloc))
	}
}

func mustGetEnv(v string) string {
	val := os.Getenv(v)
	if val == "" {
		panic(fmt.Sprintf("%s unset", v))
	}
	return val
}

//Main server which sends the traces to NRone
func Serve() {
	rand.Seed(time.Now().UnixNano())
	var err error
	Stub := &Stub{}

	fmt.Println("in serving1", Stub)

	//toxiproxy for git and redis to simulate up, down and slow responses
	Stub.ToxiClient = toxiproxy.NewClient("localhost:8474")
	Stub.ToxiProxy, _ = Stub.ToxiClient.CreateProxy("gituptest", "", "github.com:80")
	Stub.RedisProxy, _ = Stub.ToxiClient.CreateProxy("redis", "localhost:7379", "localhost:6379")

	fmt.Println("in serving2", Stub)

	Stub.h, err = telemetry.NewHarvester(
		telemetry.ConfigAPIKey(mustGetEnv("NEW_RELIC_INSERT_API_KEY")),
		telemetry.ConfigCommonAttributes(map[string]interface{}{
			"app.name":            "myServer",
			"host.name":           "localhost",
			"env":                 "testing",
			"service.instance.id": "New relic trial telemetry ",
		}),
		telemetry.ConfigBasicErrorLogger(os.Stderr),
		telemetry.ConfigBasicDebugLogger(os.Stdout),
		func(cfg *telemetry.Config) {
			cfg.MetricsURLOverride = os.Getenv("NEW_RELIC_METRIC_URL")
			cfg.SpansURLOverride = os.Getenv("NEW_RELIC_TRACE_URL")
			cfg.EventsURLOverride = os.Getenv("NEW_RELIC_EVENT_URL")
		},
	)
	if nil != err {
		panic(err)
	}
	databaseAttributes := map[string]interface{}{
		"db.type":             "sql",
		"db.instance":         "customers",
		"service.instance.id": "New relic trial DB",
	}
	redisAttributes := map[string]interface{}{
		"db.type":             "redis",
		"db.instance":         "trial",
		"service.instance.id": "New relic trial redis",
	}
	gitAttributes := map[string]interface{}{
		"type":                "Gitweb",
		"instance":            "proxy",
		"service.instance.id": "New relic trial git",
	}
	Stub.databaseHits = Stub.h.MetricAggregator().Count("database.cache.hits", databaseAttributes)
	Stub.databaseMisses = Stub.h.MetricAggregator().Count("database.cache.misses", databaseAttributes)
	Stub.RedisUpHits = Stub.h.MetricAggregator().Count("redis.up.hits", redisAttributes)
	Stub.RedisDownHits = Stub.h.MetricAggregator().Count("redis.down.hits", redisAttributes)
	Stub.RedisSlowHits = Stub.h.MetricAggregator().Count("redis.slow.hits", redisAttributes)
	Stub.GitDownHits = Stub.h.MetricAggregator().Count("git.down.hits", gitAttributes)
	Stub.GitUpHits = Stub.h.MetricAggregator().Count("git.up.hits", gitAttributes)

	go Stub.gatherMemStats()

	http.HandleFunc(Stub.wrapHandler("/", "simple", Stub.index))
	http.HandleFunc(Stub.wrapHandler("/fetch", "simple", Stub.fetch))
	http.HandleFunc(Stub.wrapHandler("/outbound", "simple", Stub.outbound))
	http.HandleFunc(Stub.wrapHandler("/redisup", "Redis is Up", Stub.RedisBackendUp))
	http.HandleFunc(Stub.wrapHandler("/redisdown", "Redis is Down", Stub.RedisBackendDown))
	http.HandleFunc(Stub.wrapHandler("/redisslow", "Redis is Slow", Stub.RedisBackendSlow))
	http.HandleFunc(Stub.wrapHandler("/gitupstatus", "Git is Up", Stub.GitcheckUp))
	http.HandleFunc(Stub.wrapHandler("/gitdownstatus", "Git is Down", Stub.GitcheckDown))
	http.ListenAndServe(":8000", nil)
}
