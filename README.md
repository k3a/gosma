# GoSMA

**Go Simple Monitoring and Alerting** — a single static binary that periodically probes your services, shows their state on a small web dashboard and alerts you when something breaks (or recovers).

## Features

- **Probes:** `http`, `dns`, `smtp`, `imap`, `ftp`, `socket` (tcp/unix, optional TLS), `file` and `process`
- **Alerters:** `ntfy` and `smtp` (e-mail)
- **Watchers:** a `dashboard` web UI
- Probes can be nested in groups that inherit `type`, `description` and other settings
- Run shell commands on state transitions (`exec_on_success`, `exec_on_failure`)
- Alerts fire only on OK ⇄ failure transitions; multiple alerters act as fallbacks; a startup mute (`delay`) and randomized check intervals avoid alert floods and thundering herds
- HTTP probe checks status codes, response body (regexp, invertible with a `!` prefix) and TLS certificate expiration
- Cross-compiles to Linux, macOS and Windows with no CGO

## Build

Requires Go 1.26+.

```sh
go build -o gosma .
# or build all platforms into ./bin
./build.sh
```

## Usage

```sh
./gosma config.yaml
```

The only argument is the path to the YAML config file. GoSMA runs until stopped.

## Configuration

```yaml
name: My Monitoring        # optional, shown on the dashboard (default: GoSMA)
url: https://status.example.com   # optional, URL where this install responds
delay: 30s                 # mute alerts for this long after startup
randomize_perc: 5          # randomize update intervals by +/- this percent

probes:
  website:
    type: http
    update_every: 30s
    url: https://example.com
    status_codes: [200, 301]      # default: 200
    response_match: "Welcome"     # regexp; prefix with ! to require NO match
    alert_after: 2                # alert after this many failed cycles
    exec_on_failure: ./notify-down.sh

  group:                          # groups inherit type, description, ...
    description: Infrastructure
    type: dns
    dns:
      server: 8.8.8.8
      query: example.com
      query_type: MX
    mail:
      type: smtp
      server: smtp.example.com
      tls: true

watchers:
  - type: dashboard
    listen: 127.0.0.1:8080

alerters:
  - type: ntfy
    server: https://ntfy.example.com
    channel: alerts
    username: user
    password: secret
    priority: high
    priority_ok: low
  - type: smtp
    server: smtp.example.com:587
    from: gosma@example.com
    to: me@example.com
    username: user
    password: secret
```

See [`testdata/simple.yaml`](testdata/simple.yaml) for a larger example.

### Common probe options

| Option | Description |
| --- | --- |
| `type` | Probe type (see below) |
| `description` | Optional text, shown as a tooltip on the dashboard |
| `update_every` | How often to run the check |
| `timeout` | Probe timeout (defaults to `update_every`) |
| `alert_after` | Alert after this many consecutive failed cycles |
| `exec_after` | Cycle number (counted from the state change) on which the exec commands run; default: the first cycle |
| `exec_on_success` / `exec_on_failure` | Shell command run on transition to OK / failed |

### Probe types

| Type | Options |
| --- | --- |
| `http` | `url`, `method`, `body`, `headers`, `status_code`, `status_codes`, `response_match` |
| `dns` | `server`, `query`, `query_type` |
| `smtp` / `imap` / `ftp` | `server`, `tls` |
| `socket` | `socket_type` (`tcp` default, `unix`, ...), `address`, `path` (unix), `tls`, `tls_server_name` |
| `file` | `path`, `path_not_exists`, `path_is_file`, `path_is_directory`, `older_than`, `not_older_than`, and `atime_`/`mtime_`/`ctime_`/`btime_` variants of both |
| `process` | `process` (process name) |

### Alerters

Alerters are tried in order and the first one that succeeds wins, so later ones act as fallbacks. Set `alert_always: true` on an alerter to have it notify even when an earlier alerter already succeeded.

| Type | Options |
| --- | --- |
| `ntfy` | `server`, `channel`, `username`, `password`, `priority`, `priority_ok` |
| `smtp` | `server`, `from`, `to`, `username`, `password`, `identity`, `crammd5`, `disable_tls` |

### Watchers

| Type | Options |
| --- | --- |
| `dashboard` | `listen` (address to bind, e.g. `0.0.0.0:8080`) |

> **Security note:** the dashboard has no authentication. Bind it to localhost or put it behind a reverse proxy if it shouldn't be public. Keep config files containing credentials out of version control.

## Tests

```sh
go test ./...
```
