# Signal

A lightweight Go service that receives Grafana webhook alerts and relays them
via Telegram and Matrix. It now also includes a simple web UI for managing
hosts, messaging settings, and your account.

## Run with Docker

The `Dockerfile` builds everything (templ code, Tailwind CSS, Go binary) into a
minimal image — the static assets are embedded in the binary, so the container
needs no files besides a config. CI publishes the image to
`ghcr.io/niklucky/signal` (`latest` for releases, `edge` for the main branch).

```bash
cp config.yaml.example config.yaml   # fill in your credentials
DB_PASSWORD=secret SESSION_SECRET=$(openssl rand -hex 32) docker compose up -d
```

`compose.yml` pulls the pre-built image from GHCR and starts Postgres and the
app on port `8080`. The database connection is injected via `DATABASE_URL`, so
`config.yaml` only needs the server, Telegram/Matrix, and scheduler sections.
Postgres data lives in the `db-data` volume; `config.yaml` is bind-mounted from
the working directory. `hosts.yml` is optional — place it next to
`compose.yml` and add a volume mount for it if you use the host scheduler.
Use `docker compose pull && docker
compose up -d` to update to a newer image. For staging, switch the image tag to
`edge` to track the main branch.

To build the image yourself instead:

```bash
docker build -t signal .
docker run -p 8080:8080 -v "$PWD/config.yaml:/etc/signal/config.yaml" signal
```

To use a locally built image with compose, override `SIGNAL_IMAGE`:

```bash
SIGNAL_IMAGE=signal:latest docker compose up -d
```

## Quick start

1. Copy the example config and fill in your credentials:

   ```bash
   cp config.yaml.example config.yaml
   ```

2. Start Postgres (for example with Docker):

   ```bash
   docker run -d --name signal-db \
     -e POSTGRES_USER=signal \
     -e POSTGRES_PASSWORD=signal \
     -e POSTGRES_DB=signal \
     -p 5432:5432 \
     postgres:17-alpine
   ```

3. Build the UI assets and the server:

   ```bash
   make build
   ```

   This downloads `templ` and `tailwindcss` into `bin/`, generates Go code from
   `*.templ` files, builds `web/static/css/main.css`, and compiles the `signal`
   binary.

4. Run the server:

   ```bash
   ./signal -config config.yaml
   ```

5. Open the UI at `http://localhost:8080` and sign in with the default user
   (`default@signal.local`). You will be asked to set a password on first login.

6. Point Grafana webhook notifications to:

   ```
   http://<host>:8080/webhooks/grafana
   ```

## Configuration

| Section     | Field          | Description                         |
|-------------|----------------|-------------------------------------|
| `server`    | `address`      | HTTP listen address                 |
| `database`  | `url`          | Postgres connection URL             |
| `scheduler` | `hosts_file`   | Path to the hosts YAML file         |
| `telegram`  | `enabled`      | Enable Telegram relay               |
| `telegram` | `bot_token`    | Telegram bot token                  |
| `telegram` | `chat_id`      | Target Telegram chat ID             |
| `telegram` | `proxy_url`    | Optional Telegram API proxy host    |
| `matrix`   | `enabled`      | Enable Matrix relay                 |
| `matrix`   | `homeserver`   | Matrix homeserver URL               |
| `matrix`   | `user_id`      | Bot user ID                         |
| `matrix`   | `access_token` | Bot access token                    |
| `matrix`   | `room_id`      | Target room ID                      |

## Web UI

The web UI is built with [templ](https://templ.guide/) (typed Go templates),
[Tailwind CSS](https://tailwindcss.com/), and [htmx](https://htmx.org/). Static
assets from `web/static/` are embedded into the binary at build time, just like
the compiled `templ` components.

Useful commands:

```bash
make deps       # install templ and tailwindcss CLIs
make templ      # generate Go code from *.templ files
make css        # build web/static/css/main.css
make build      # generate assets and compile the binary
make dev        # build assets and run the server
```

For production, set `SESSION_SECRET` to a strong random value:

```bash
SESSION_SECRET=$(openssl rand -hex 32) ./signal -config config.yaml
```

Screens:

- `/` — home with sign-in link
- `/login` — sign in
- `/dashboard` — response-time charts and host status
- `/hosts` — list, add, edit, and delete monitored hosts
- `/messaging` — edit Telegram/Matrix config (writes `config.yaml`; restart required)
- `/profile` — change email and password

## What it does

## Host scheduler

Signal can also periodically check HTTP endpoints configured in `hosts.yml` (path is set via `scheduler.hosts_file`, default: `hosts.yml`).

Example `hosts.yml`:

```yaml
hosts:
  - name: "example-api"
    method: "GET"
    url: "https://example.com/health"
    headers:
      Authorization: "Bearer token"
    timeout: 10
    interval: 60
    resend_interval: 300

  - name: "example-login"
    method: "POST"
    url: "https://example.com/api/login"
    headers:
      Content-Type: "application/json"
    body: '{"username":"test","password":"test"}'
    timeout: 15
    interval: 120
    resend_interval: 600
```

Host fields:

| Field             | Description                                            |
|-------------------|--------------------------------------------------------|
| `name`            | Display name for logs and alerts                       |
| `method`          | HTTP method; defaults to `GET`                         |
| `url`             | Full URL to request                                    |
| `headers`         | Optional request headers                               |
| `body`            | Optional JSON request body                             |
| `timeout`         | Request timeout in seconds; defaults to `10`           |
| `interval`        | Seconds between checks                                 |
| `resend_interval` | Seconds before re-sending an alert while still failing |

When a check returns a non-`200` status or fails to connect, Signal sends an alert via Telegram and/or Matrix. The alert is re-sent only after `resend_interval` while the host keeps failing. Once the host returns `200`, a recovery message is sent.

Every check result is stored in the configured Postgres database in the `events` table.

On each incoming Grafana webhook:

- Parses the Grafana alert JSON.
- Renders a short, readable message from the first alert.
- Sends the message to Telegram (HTML formatting) and/or Matrix (Markdown).

### Example rendered message

```text
🔥 [FIRING:1] CPU is high load

Status: firing
Instance: node_exporter:9100
Values:
  A = 4.17
  C = 1

Summary:
High CPU load on node_exporter:9100

Since: 2026-06-22T15:13:30Z

View in Grafana
Silence alert
```

When `status` is `resolved`, the emoji becomes ✅.

## Debugging

Set `LOG_LEVEL=debug` to print the full Grafana JSON payload to the console:

```bash
LOG_LEVEL=debug go run ./cmd/server -config config.yaml
```

Supported levels: `debug`, `info`, `warn`, `error`. Default is `info`.
