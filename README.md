# iplist-for-mikrotik

HTTP proxy for [iplist.opencck.org](https://iplist.opencck.org/) that forwards query parameters unchanged and returns a deduplicated MikroTik script.

When fetching large IP lists, the upstream service may return the same CIDR subnet multiple times (once per associated domain or site). MikroTik address lists do not need repeated entries for the same subnet. This proxy removes duplicate subnet lines while keeping all other content intact.

## Features

- Transparent query forwarding to the upstream API
- Deduplication of CIDR subnet lines (first occurrence is kept)
- Domain names and non-CIDR lines are not deduplicated
- Response served as a downloadable plain-text file (`iplist.rsc`)
- Per-request statistics logged to stdout

## Requirements

- Go 1.23 or later

## Build

```bash
go build -o iplist-proxy .
```

## Run

```bash
./iplist-proxy
```

By default the server listens on `:8090` and proxies requests to `https://iplist.opencck.org/`.

## Configuration

| Variable       | Default                         | Description              |
|----------------|---------------------------------|--------------------------|
| `LISTEN_ADDR`  | `:8090`                         | Address and port to bind |
| `UPSTREAM_URL` | `https://iplist.opencck.org/`   | Upstream API base URL    |

Example:

```bash
LISTEN_ADDR=:8080 UPSTREAM_URL=https://iplist.opencck.org/ ./iplist-proxy
```

## Usage

Send a GET request with the same query parameters you would use on the upstream service. They are forwarded as-is.

**Client request:**

```
http://localhost:8090/?format=mikrotik&data=cidr4&site=site.com&template=cidr4_to_firevol_list
```

**Proxied upstream request:**

```
https://iplist.opencck.org/?format=mikrotik&data=cidr4&site=site.com&template=cidr4_to_firevol_list
```

Download the result directly:

```bash
curl -o iplist.rsc "http://localhost:8090/?format=mikrotik&data=cidr4&template=cidr4_to_firevol_list"
```

Refer to the [iplist.opencck.org](https://iplist.opencck.org/) documentation for available `format`, `data`, `site`, `template`, and other parameters.

## Deduplication

Only lines containing an IPv4 CIDR notation (e.g. `1.2.3.0/24`) are checked for duplicates. When the same subnet appears more than once, subsequent lines are dropped.

Lines without a CIDR address — including MikroTik header commands, delays, and domain entries — are always preserved.

Example:

```
add list=test address=1.2.3.0/24 comment=site1   # kept
add list=test address=4.5.6.0/24 comment=site2   # kept
add list=test address=1.2.3.0/24 comment=site3   # removed (duplicate subnet)
add list=test address=youtube.com comment=site4  # kept (not a subnet)
add list=test address=youtube.com comment=site5  # kept (domains are not deduplicated)
```

## Request logging

Each request is logged to the console with timing and deduplication stats:

```
[127.0.0.1:54321] format=mikrotik&data=cidr4&template=cidr4_to_buedpi | 200 | 1.2s | lines: 12203 -> 8663 (removed 3540) | 512000 bytes
```

Failed requests and non-200 upstream responses are logged as well.

## Tests

```bash
go test ./...
```

## License

MIT
