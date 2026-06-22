# Signal

A lightweight Go service that receives Grafana webhook alerts and relays them
via Telegram and Matrix.

## Quick start

1. Copy the example config and fill in your credentials:

   ```bash
   cp config.yaml.example config.yaml
   ```

2. Run the server:

   ```bash
   go run ./cmd/server -config config.yaml
   ```

3. Point Grafana webhook notifications to:

   ```
   http://<host>:8080/webhooks/grafana
   ```

## Configuration

| Section     | Field          | Description                         |
|-------------|----------------|-------------------------------------|
| `server`    | `address`      | HTTP listen address                 |
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

When a check returns a non-`200` status or fails to connect, Signal sends an alert via Telegram and/or Matrix. The alert is re-sent only after `resend_interval` while the host keeps failing.

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
