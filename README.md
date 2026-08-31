# Weather Service

HTTP server that returns the current forecast and temperature classification for a given location using the [National Weather Service API](https://www.weather.gov/documentation/services-web-api).

## Requirements

- Go 1.27+ — install from https://go.dev/dl/

## Build check

```sh
make build-check
```

## Test

```sh
make test
```

`internal/api` and `internal/routes` use **mocks**: `.On(...)`, `.Once()`, and `AssertExpectations` check that the forecaster was called as expected. `internal/nws` uses a **stub** — it replays fixed responses, and the checks are on the decoded `*forecast.Forecast`.

### Fixtures

Captured NWS responses live in `internal/nws/testdata/`, embedded with `//go:embed`.

- Payloads are served as **raw bytes**, never marshalled from `pointsResponse`/`forecastResponse`. Encoding with the same structs the client decodes into would let a wrong `json` tag agree with itself and pass.
- `fetch` duplicates its status and decode handling across a debug branch (buffers the body) and an info branch (streams it), so `TestGetForecast_Errors` runs every case at both `slog.LevelInfo` and `slog.LevelDebug`.

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

### Optional: request file

[`api.http`](api.http) holds the same endpoints with assertions attached, so a run
reports pass/fail rather than printing a body to read. It covers what the curl
examples above do not: all five `400` validation messages, the `404` for a valid
coordinate outside NWS coverage, and the `405`/`404` chi returns for an unclaimed
verb and an unrouted path. Not required — everything the project needs runs from
`make`.

The format is not tied to one editor. Any JetBrains IDE runs it from the gutter, and
VS Code does too with the
[httpYac](https://marketplace.visualstudio.com/items?itemName=anweber.vscode-httpyac)
extension. Headless, for CI:

```sh
make run &
npx httpyac send api.http --all
```

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

## Status

Version 0.4 — work in progress.