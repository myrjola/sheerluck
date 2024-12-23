# Sheerluck

AI-powered murder mystery game

## Quickstart

### Start go server

Make sure you're using the go version configured in `go.mod`. To start the server, run the following:

```
go run ./cmd/web/
```

Navigate to http://localhost:4000 to see the service in action.

## Operations

### Deploying

This project uses [Fly.io](https://fly.io/) for infrastructure and [Litestream](https://litestream.io/) for [SQLite](https://www.sqlite.org/) database backups. It's a single instance Dockerized application with a persistent volume. Try `fly launch` to configure your own. You might also need to add some secrets to with `fly secrets`.

### Database access

The container image contains sqlite3 binary to make it easy to manipulate the live production database.

```sh
fly ssh console -s --pty --user sheerluck -C "/dist/sqlite3 /data/sheerluck.sqlite3"
```


### Recovering database

One way to recover a lost or broken database is to restore it with Litestream. The process could still use some improvements but at least it works. Notably, you need to have a working machine running so that you can run commands on it. Another alternative is to clone the machine with an empty volume and populate it yourself using the `fly sftp shell` command.

```
# list databases
fly ssh console -s --user sheerluck -C "/dist/litestream databases"
# list snapshot generations of selected database
fly ssh console -s --user sheerluck -C "/dist/litestream snapshots /data/sheerluck.sqlite3"
# restore latest snapshot to /data/sheerluck4.sqlite
fly ssh console -s --user sheerluck -C "/dist/litestream restore -o /data/sheerluck4.sqlite -generation db5b998e60a203a3 /data/sheerluck.sqlite3"

# Edit fly.toml env SHEERLUCK_SQLITE_URL = "/data/sheerluck4.sqlite" before deploying to take new database into use
vim fly.toml

# Deploy the new configuration
fly deploy
```

### Performance investigation

Use [pprof](https://pkg.go.dev/net/http/pprof) for perfomance investigation. You should open a Wireguard VPN following the [fly.io documentation](https://fly.io/docs/networking/private-networking/).

```
# look up the IPv6 address of the server check the DNS configuration from your Wireguard configuration
dig +short aaaa sheerluck.internal @fdaa:4:7523::3
# gather 30-second CPU profile and open it in web browser.
go tool pprof -http=: "http://fdaa:4:7523:a7b:359:fb0b:a4b0:2:6060/debug/pprof/profile?seconds=30"
```

## Attribution

Sheerluck logo made by Martin Yrjölä.

Images was created with the assistance of [DALL·E 2](https://openai.com/dall-e-2) and [DALL·E 2](https://openai.com/dall-e-3).

HeroIcons made by [steveschoger](https://twitter.com/steveschoger). Available on https://heroicons.dev/.

Game Icons made by [Delapouite](https://delapouite.com/) and Skoll. Available on https://game-icons.net.
