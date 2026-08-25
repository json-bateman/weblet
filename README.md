# Webadelphos (Web Brother)

A tiny Cockpit-style system monitor for **web servers** specifically: a single Go binary that serves a live
web view of the machine it runs on — host facts and the running process list — pushed to the browser over server-sent events.
Configured to watch for Quadlet Unit files because I use podman with `.container`s.

[chi]: https://github.com/go-chi/chi
[Datastar]: https://data-star.dev
[templ]: https://templ.guide

## Run

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

Then open **http://localhost:44223**. Config is `WEBADELPHOS_`-prefixed (see
`env.go`); override via `.env`.

## Development on macOS (Lima VM)

These files only exist on Linux, so run inside a linux VM, I recommend lima. The repo's port 44223 needs
forwarded to the host:

```bash
limactl shell centos10 sh -c 'cd /home/lima/centos10/webadelphos && go run ./cmd'
```

Then open **http://localhost:44223** on the Mac.

## Viewing a remote server from your Mac

If `webadelphos` is running on a server you SSH into, forward the port
over SSH instead of exposing it to the network:

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
