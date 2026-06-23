# Learn Cooling

An interactive explainer for how a home air conditioner works. Click through the
four phases of the refrigeration cycle — **compression → condensation → expansion
→ evaporation** — and see what happens to the refrigerant at each step and what it
means for your home. You can also:

- flip the loop into **heat-pump (heating) mode** to see the reversing-valve idea,
- switch the **refrigerant** (R-410A, R-32, R-134a, and R-717 / ammonia) and watch
  the real pressures and discharge temperatures change, and
- browse **"what could go wrong"** — the common home-AC faults (dirty filter,
  refrigerant leak, dirty/frozen coils, weak compressor…) tied to the phase each one
  disrupts.

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

The data — phases, modes, refrigerants, fans, and faults — is defined once as Go
structs in `main.go`. It drives the server-rendered page, the clickable diagram, and the
`/api/cycle` JSON endpoint. Pressures and temperatures aren't stored per phase;
the browser computes them from the selected refrigerant's saturation curve and the
current mode.

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
| `/`            | The interactive page                          |
| `/api/cycle`   | Full dataset: modes, refrigerants, phases, faults |
| `/api/phases`  | Just the four phases (kept for compatibility) |
| `/healthz`     | Health check (used by Render)                 |
| `/static/*`    | Embedded CSS / JS                             |

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
