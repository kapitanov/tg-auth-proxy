# tg-auth-proxy

A reverse proxy with built-in authentication via [Telegram Login Widget](https://core.telegram.org/widgets/login).
Only specified Telegram users will have access to backend site.

## Usage

1. Clone repository to any convenient location, e.g. `/opt/tg-auth-proxy`.
2. Create a `.env` file containing app parameters (see below)
3. Edit `docker-compose.yaml` file if neccessary.
4. Build and run an application:

   ```shell
   docker compose build
   docker compose up -d 
   ```

## Parameters

This app is configured by env variables only.

| Variable name      | Default value  | Description                         |
| ------------------ | -------------- | ----------------------------------- |
| `TG_ALLOWED_USERS` | **Required**   | List of Telegram user id/names/urls |
| `TG_BOT_TOKEN`     | **Required**   | Telegram bot access token           |
| `BACKEND_URL`      | **Required**   | Upstream URL                        |
| `LISTEN_ADDR`      | `0.0.0.0:8000` | Endpoint to listen                  |

`TG_ALLOWED_USERS` should contain a list of Telegram users in the following format:

* Each user might be specified by his/hers ID, username, username with `@` prefix or a `t.me` URL:

  * `123456`
  * `username`
  * `@username`
  * `http://t.me/username`
  * `https://t.me/username`

* Users should be separated by space, comma or semicolon
* Leading and trailing spaces are ignored

## License

[MIT](LICENSE)
