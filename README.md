# Weather Service

HTTP server that returns the current forecast and temperature classification for a given location using the [National Weather Service API](https://www.weather.gov/documentation/services-web-api).

## Requirements

- Go 1.26+ — install from https://go.dev/dl/

## Build check

```sh
make build-check
```

## Test

```sh
make test
```

## Run

```sh
make run
```

Server starts on port 8080 by default. Override with:

```sh
make run PORT=9090
```

## Debug logging

Logs full NWS raw response bodies at DEBUG level:

```sh
make run-debug        # raw JSON output
make run-debug-jq     # pretty-printed via jq (requires jq installed)
```


## Test the endpoint

**Plano, TX:**

```sh
curl "http://localhost:8080/weather?lat=33.019844&lon=-96.698883"
```

**El Segundo, CA** (2 hours behind Plano, TX — good edge case):

```sh
curl "http://localhost:8080/weather?lat=33.9192&lon=-118.4165"
```

**Fairbanks, AK** (extreme cold edge case):

```sh
curl "http://localhost:8080/weather?lat=64.8401&lon=-147.7164"
```

Response:

```json
{
  "forecast": "Mostly Cloudy",
  "temperature_classification": "moderate"
}
```

- `forecast` — short forecast string from NWS (`periods[0].shortForecast`)
- `temperature_classification` — `very hot` (>=100°F), `hot` (>=85°F), `cold` (<=55°F), `very cold` (<=30°F), or `moderate`
- `periods[0]` is always the active NWS period for the location's local time — pre-dawn returns "Overnight", not "Today"

## Rate limiting

Per-IP rate limiter: 5 req/sec, burst of 10. Start the server first (`make run`), then fire 20 concurrent requests against `/health` (instant response, no NWS latency) to observe the burst limit:

```sh
for i in $(seq 1 20); do curl -s -o /dev/null -w "%{http_code}\n" "http://localhost:8080/health" & done; wait
```

First 10 return `200`, remainder return `429 Too Many Requests`.

Also, observable with the actual weather endpoint (NWS latency spreads responses but 429s still appear):

```sh
for i in $(seq 1 20); do curl -s -o /dev/null -w "%{http_code}\n" "http://localhost:8080/weather?lat=33.019844&lon=-96.698883" & done; wait
```

## Health check

```sh
curl http://localhost:8080/health
```


## Development

Enable pre-commit hooks (Go version check, fmt, fix-diff, lint):

```sh
git config core.hooksPath .githooks
```

Run checks manually:

```sh
make fmt-diff   # show formatting diff
make fix-diff   # show go fix changes
make lint       # golangci-lint
```

## Prod Improvements

- [ ] Using `slog` for now: Forward `slog` JSON stdout to Datadog Logs or Grafana/Loki (needs log agent: Datadog Agent).
- [ ] Using chi `RequestID` for now: replace with `dd-trace-go` to stamp `dd.trace_id` on log lines and link them to Datadog APM traces.
- [ ] Caching: two-TTL cache for NWS responses (points: 24h, forecast: 1h) backed by Redis to reduce latency and upstream calls.