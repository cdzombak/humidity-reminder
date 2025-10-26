# humidity-reminder

## What It Does

`humidity-reminder` watches the National Weather Service (weather.gov) forecast for your location, calculates the recommended indoor relative humidity for the coming week, and emails you via Mailgun whenever that recommendation changes. It keeps a tiny JSON state file so you only get notified when there's something new to do (for example, when the median overnight low drops and you should lower your humidifier setting).

## Installation & Setup

### macOS via Homebrew

```shell
brew install cdzombak/oss/humidity-reminder
```

### Debian via Apt repository

Install my Debian repository if you haven't already:

```shell
sudo apt-get install ca-certificates curl gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://dist.cdzombak.net/deb.key | sudo gpg --dearmor -o /etc/apt/keyrings/dist-cdzombak-net.gpg
sudo chmod 0644 /etc/apt/keyrings/dist-cdzombak-net.gpg
echo -e "deb [signed-by=/etc/apt/keyrings/dist-cdzombak-net.gpg] https://dist.cdzombak.net/deb/oss any oss\n" | sudo tee -a /etc/apt/sources.list.d/dist-cdzombak-net.list > /dev/null
sudo apt update
```

Then install `humidity-reminder` via `apt`:

```shell
sudo apt install humidity-reminder
```

### Manual installation from build artifacts

Pre-built binaries for Linux and macOS on multiple architectures are attached to every [GitHub Release](https://github.com/cdzombak/humidity-reminder/releases). Debian packages for each release are published as well.

### Build and install locally

```shell
git clone https://github.com/cdzombak/humidity-reminder.git
cd humidity-reminder
make build

cp out/humidity-reminder $INSTALL_DIR
```

### Docker image

Multi-architecture Docker images are published to [Docker Hub](https://hub.docker.com/r/cdzombak/humidity-reminder) and [GHCR](https://github.com/cdzombak/humidity-reminder/pkgs/container/humidity-reminder). Images use `scratch` as the final stage and include only the compiled binary plus CA certificates.

Example run:

```shell
docker run --rm \
  -v /home/cdzombak/.config/humidity-reminder/config.yaml:/app/config.yaml:ro \
  -v /var/lib/humidity-reminder:/var/lib/humidity-reminder \
  cdzombak/humidity-reminder:1 \
  -config /app/config.yaml
```

Ensure the `state_dir` in your config points to the in-container path (e.g., `/var/lib/humidity-reminder`) that you've bind-mounted from the host.

## Configuration

Copy `config.example.yaml` to a private location, fill in your details, and point `humidity-reminder` at it with `-config`:

```yaml
latitude: 41.1234
longitude: -81.5679
weather:
  user_agent: "humidity-reminder/1.0 (you@example.com)"
  timeout: "10s"
mailgun:
  domain: "mg.example.com"
  api_key: "key-XXXXXXXXXXXXXXXXXXXXXXX"
  from: "Humidity Reminder <humidity@example.com>"
  to: "you@example.com"
state_dir: "/var/lib/humidity-reminder"
```

Key settings:

- **latitude / longitude**: Decimal degrees (WGS84). These drive the weather.gov grid lookup.
- **weather.user_agent**: NOAA requires a descriptive User-Agent with contact info. Use something like `humidity-reminder/1.0 (name@example.com)`.
- **weather.timeout**: Go-style duration (default `10s`) for weather API requests.
- **mailgun.domain / api_key / from / to**: Credentials and addressing for Mailgun's HTTP API. The program sends a single email whenever the recommendation changes.
- **state_dir**: Absolute path where a `state.json` file is stored. It should be writable by the process and persisted across runs.

## Usage

Run the program with your config file:

```shell
humidity-reminder -config /etc/humidity-reminder/config.yaml
```

On each run the program:

1. Fetches the latest forecast periods from weather.gov for your coordinates.
2. Extracts the next seven overnight low temperatures.
3. Computes the median overnight low, then converts it to a recommended indoor humidity using `libwx`.
4. Compares that recommendation with the previous value stored in `state_dir/state.json`.
5. Sends a Mailgun email if the recommendation changed, then stores the new value and timestamp.

Available flags:

- `-config string` (required): Path to your YAML configuration file.
- `-version`: Print build version information and exit.
- `-help`: Standard Go flag help output.

### Cron Example

Run `humidity-reminder` daily to check for changes:

```text
0 6 * * * /usr/bin/humidity-reminder -config /etc/humidity-reminder/config.yaml
```

Make sure your `state_dir` is writable by whatever user runs the job so the program can remember the last recommendation it sent.

### launchd Example (macOS)

Create `~/Library/LaunchAgents/net.cdzombak.humidity-reminder.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>net.cdzombak.humidity-reminder</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/humidity-reminder</string>
        <string>-config</string>
        <string>/Users/YOUR_USERNAME/.config/humidity-reminder/config.yaml</string>
    </array>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>
        <integer>6</integer>
        <key>Minute</key>
        <integer>0</integer>
    </dict>
    <key>StandardErrorPath</key>
    <string>/tmp/humidity-reminder.err</string>
    <key>StandardOutPath</key>
    <string>/tmp/humidity-reminder.out</string>
</dict>
</plist>
```

Load the job with:

```shell
launchctl load ~/Library/LaunchAgents/net.cdzombak.humidity-reminder.plist
```

## License

GNU General Public License v3.0. See `LICENSE` in this repository.

## About

- Issues: [github.com/cdzombak/humidity-reminder/issues](https://github.com/cdzombak/humidity-reminder/issues)
- Author: [Chris Dzombak](https://www.dzombak.com) ([GitHub @cdzombak](https://github.com/cdzombak))
