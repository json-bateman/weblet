# centos-streamed

A tiny Cockpit-style **system monitor**: a single Go binary that serves a live
web view of the machine it runs on — host facts and the running process list —
pushed to the browser over server-sent events.

No database, no reverse proxy, no deploy tooling. Just a webserver reading
`/proc`.

## How it works

`cmd/main.go` starts a [chi] HTTP server. The home page renders a host-info card
and a process table; a `data-init` [Datastar] attribute opens an SSE connection
to `/sse`, which every two seconds re-reads `/proc` and patches both fragments
back into the page live.

- **Host facts** — hostname, `PRETTY_NAME` from `/etc/os-release`, kernel from
  `/proc/sys/kernel/osrelease`, plus memory/uptime from `/proc` and CPUs/arch
  from the Go runtime.
- **Processes** — walks `/proc/<pid>/{status,stat}` for each process: command,
  resolved user, state, threads, resident memory and cumulative CPU time,
  sorted by memory (top 40).

These reads only return data on **Linux**, so run it inside the VM, not on macOS.

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

Then open **http://localhost:44223**. Config is `STREAMED_`-prefixed (see
`env.go`); override via `.env`.

## Development on macOS (Lima VM)

`/proc` only exists on Linux, so run inside the `centos10` VM. The repo is
mounted at `/home/lima/centos10/centos-streamed` and guest port 44223 is
forwarded to the host:

```bash
limactl shell centos10 sh -c 'cd /home/lima/centos10/centos-streamed && go run ./cmd'
```

Then open **http://localhost:44223** on the Mac.

## Viewing a remote server from your Mac

If `centos-streamed` is running on a server you SSH into, forward the port
over SSH instead of exposing it to the network:

```bash
ssh -L 44223:localhost:44223 user@your-server
```

Then open **http://localhost:44223** on your Mac. Add `-N -f` to run the
tunnel in the background, or put a `LocalForward` entry in `~/.ssh/config`
to avoid retyping the command:

```
Host mycentos
    HostName your-server
    User user
    LocalForward 44223 localhost:44223
```

then just `ssh mycentos`.
