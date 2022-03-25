This module starts a server at 8000 on localhost which exposes the below endpoints

/redisup -> requests sent to this endpoint will be proxied to a localhost redis which is up\n
/redisdown -> requests sent to this endpoint will be proxied to a localhost redis which is down\n
/redisslow -> requests sent to this endpoint will be proxied to a localhost redis which has a slow response\n


To simulate a redis which is down or slow, we use toxiproxy package which can simulate connection down and add latencies to the request

/gitupstatus -> simulates git up
/gitdownstatus -> simulates git down

All the endpoints are instrumented via the New Relic Telemetry Go SDK.

The below ENV variables must be set up prior to starting the server.
NEW_RELIC_EVENT_URL
NEW_RELIC_TRACE_URL
NEW_RELIC_METRIC_URL
NEW_RELIC_INSERT_API_KEY

Starting the server
From main platform folder, execute go run main.go
