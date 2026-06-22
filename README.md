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

| Section    | Field          | Description                         |
|------------|----------------|-------------------------------------|
| `server`   | `address`      | HTTP listen address                 |
| `telegram` | `enabled`      | Enable Telegram relay               |
| `telegram` | `bot_token`    | Telegram bot token                  |
| `telegram` | `chat_id`      | Target Telegram chat ID             |
| `telegram` | `proxy_url`    | Optional Telegram API proxy host    |
| `matrix`   | `enabled`      | Enable Matrix relay                 |
| `matrix`   | `homeserver`   | Matrix homeserver URL               |
| `matrix`   | `user_id`      | Bot user ID                         |
| `matrix`   | `access_token` | Bot access token                    |
| `matrix`   | `room_id`      | Target room ID                      |

## What it does

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
