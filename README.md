# Weblet

A tiny Linux system monitor for **Web Servers**.  
This app is a single Go binary that serves a live web view of the machine it runs on.

Specifially, this app is for people that use:

1. [Caddy](https://github.com/caddyserver/caddy) - reverse proxy / web server  
2. [Quadlets](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html) - Podman's systemd integration
3. Static sites via index.html under /var/www

## Installation

```bash
curl -sL https://raw.githubusercontent.com/json-bateman/weblet/main/install.sh | sh
```
Downloads the latest Linux release for your CPU architecture and starts it in the
background. Open **http://localhost:44223**.

## Viewing a remote server via SSH

If `weblet` is running on a server you SSH into, forward the port over SSH instead of exposing it to the network.

```bash
ssh -L 44223:localhost:44223 user@your-server
```

Then open **http://localhost:44223** on your local machine. Add `-N -f` to run the
tunnel in the background, or put a `LocalForward` entry in `~/.ssh/config`
to avoid retyping the command:

```
Host myserver
    HostName your-server-hostname
    User user
    LocalForward 44223 localhost:44223
```

then just `ssh myserver`.

## Local Development

Dev loop with hot reload (regenerates templ + rebuilds on save):

```bash
task setup   # once: install pinned templ / air tools into go.mod
task         # runs air
```

Or build and run directly:

```bash
task go:build        # -> ./release/web
go run ./cmd
```

Then open **http://localhost:44223**. Config is `WEBLET_`-prefixed (see
`env.go`); override via `.env`.

## Development on MacOS (Lima VM)

Run inside a linux VM, I recommend lima.

```bash
limactl shell centos10 sh -c 'cd /home/lima/centos10/weblet && task'
```

Then open **http://localhost:44223** on the Mac.

### The Stack

[chi](https://github.com/go-chi/chi) - HTTP router  
[Datastar](https://data-star.dev) - hypermedia/reactivity over SSE  
[templ](https://templ.guide) - HTML templating
