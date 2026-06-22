# Signal

A lightweight Go service that receives Grafana webhook alerts and relays them
via Telegram (Matrix support is planned for the next iteration).

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
   http://<host>:8080/webhook
   ```

## Configuration

| Section    | Field         | Description                         |
|------------|---------------|-------------------------------------|
| `server`   | `address`     | HTTP listen address                 |
| `telegram` | `enabled`     | Enable Telegram relay               |
| `telegram` | `bot_token`   | Telegram bot token                  |
| `telegram` | `chat_id`     | Target Telegram chat ID             |
| `matrix`   | `enabled`     | Enable Matrix relay (reserved)      |
| `matrix`   | `homeserver`  | Matrix homeserver URL               |
| `matrix`   | `user_id`     | Bot user ID                         |
| `matrix`   | `access_token`| Bot access token                    |
| `matrix`   | `room_id`     | Target room ID                      |

## What it does

On each incoming Grafana webhook:

- Reads the full HTTP request (headers + body).
- Pretty-prints the payload as an HTML Telegram message.
- Sends the dump to the configured Telegram chat.
