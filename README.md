# live-pprof

`live-pprof` is a desktop pprof monitor built with `Wails + Vue 3 + TypeScript + Vuetify`.

It keeps the original Go-side profile fetching and parsing logic, but the old browser-hosted Next.js frontend has been refactored into a Wails desktop app with a Vue/Vuetify UI.

## Stack

- Go backend for fetching and parsing pprof data
- Wails desktop shell
- Vue 3 + TypeScript frontend
- Vuetify UI components
- ECharts timeline charts

## Run In Dev

```bash
wails dev
```

This starts the desktop shell and the frontend dev server together.

## Build

```bash
wails build
```

The packaged Windows executable is generated at [build/bin/live-pprof.exe](/E:/work/mgap/build/bin/live-pprof.exe).

## Usage

#### Step 1: expose pprof endpoints in your Go app

```go
package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	log.Println(http.ListenAndServe("localhost:6060", nil))
}
```

#### Step 2: open `live-pprof` and enter one of these

```text
6060
localhost:6060
http://localhost:6060/debug/pprof
```

The desktop app normalizes the input to the base pprof endpoint and can:

- Detect the main `/debug/pprof` endpoint plus heap, CPU, allocs and goroutine sub-endpoints
- Sample heap, CPU, allocs and goroutine profiles on an interval
- Visualize top functions over time with ECharts
- Switch between live data and embedded mock data

## Notes

- Metrics history is kept in window memory and resets when the desktop app reloads.
- Large retained-sample counts still make charts heavier to render.
- This is intended for local development and quick inspection, not as a replacement for Prometheus and Grafana.
