# Learn Cooling

An interactive explainer for how a home air conditioner works. Click through the
four phases of the refrigeration cycle — **compression → condensation → expansion
→ evaporation** — and see what happens to the refrigerant at each step and what it
means for your home.

Live at [learncooling.com](https://learncooling.com).

## Stack

A single self-contained Go web server. There's no build step and no Node — the HTML,
CSS, JS, and the SVG diagram are embedded into the binary with `//go:embed`.

```
main.go                 # server, routing, and the phase data (source of truth)
web/templates/index.html
web/static/style.css
web/static/app.js
```

The four phases are defined once as Go structs in `main.go`. They drive the
server-rendered page, the clickable diagram, and the `/api/phases` JSON endpoint.

## Run locally

```bash
make dev          # go run .
# or
make builds && make run
```

Then open http://localhost:8080. Set `PORT` to change the port.

## Routes

| Path           | Description                          |
| -------------- | ------------------------------------ |
| `/`            | The interactive page                 |
| `/api/phases`  | The four phases as JSON              |
| `/healthz`     | Health check (used by Render)        |
| `/static/*`    | Embedded CSS / JS                    |

## Deploy

Hosted on Render as a **native Go** web service. Pushing to `master` auto-deploys.

Render service settings:

| Setting        | Value                  |
| -------------- | ---------------------- |
| Runtime        | Go                     |
| Build Command  | `make builds`          |
| Start Command  | `make run`             |
| Health Check   | `/healthz`             |

There's no build step beyond `go build` — the web assets are embedded in the
binary, and the server binds to the `PORT` Render provides.
