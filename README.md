# Sheerluck

AI-powered murder mystery game

## Quickstart

### Install dependencies and configure linting

```
make init
```

### Start go server

```
make dev
```

Navigate to http://localhost:4000 to see the service in action. You
can [attach a debugger](https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html) to it.

## Operations

### Select which Fly app is targeted.

If you get something like the following error when running the below commands:

```
Error: the config for your app is missing an app name, add an app field to the fly.toml file or specify with the -a flag
```

Then, you need to select the fly app you have deployed:

```
export FLY_APP=sheerluck
```

### Deploying

This project uses [Fly.io](https://fly.io/) for infrastructure and [Litestream](https://litestream.io/)
for [SQLite](https://www.sqlite.org/) database backups. It's a single instance Dockerized application with a persistent
volume. Try `fly launch` to configure your own. You might also need to add some secrets to with `fly secrets`.

### Database access

The container image contains sqlite3 binary to make it easy to manipulate the live production database.

```sh
make fly-sqlite3
```

### Recovering database

One way to recover a lost or broken database is to restore it with Litestream. The process could still use some
improvements but at least it works. Notably, you need to have a working machine running so that you can run commands on
it. Another alternative is to clone the machine with an empty volume and populate it yourself using the `fly sftp shell`
command.

```
# list databases
fly ssh console --user sheerluck -C "/dist/litestream databases"
# list snapshot generations of selected database
fly ssh console --user sheerluck -C "/dist/litestream snapshots /data/sheerluck.sqlite3"
# restore latest snapshot to /data/sheerluck4.sqlite
fly ssh console --user sheerluck -C "/dist/litestream restore -o /data/sheerluck4.sqlite -generation db5b998e60a203a3 /data/sheerluck.sqlite3"

# Edit fly.toml env SHEERLUCK_SQLITE_URL = "/data/sheerluck.sqlite3" before deploying to take new database into use
vim fly.toml

# Deploy the new configuration
fly deploy
```

### Performance investigation

Use [pprof](https://pkg.go.dev/net/http/pprof) for perfomance investigation.

Proxy the pprof server to your local machine.

```sh
fly proxy 6060:6060
```

Capture a CPU profile of the running app.

```sh
go tool pprof --http=: "http://localhost:6060/debug/pprof/profile?seconds=30"
```

Capture a goroutine stack traces.

```sh
go tool pprof -top "http://localhost:6060/debug/pprof/goroutine"
```

### CI/CD and preview environments

This project uses [GitHub Actions](https://docs.github.com/en/actions) for CI/CD.

### Creating new deployment

Prerequisite: Ensure you have [Fly](https://fly.io/docs/) set up correctly with `fly auth whoami`.

Create a new app with a globally unique name.

```sh
fly apps create sheerluck-staging
```

Create a bucket for the database backups. This should configure the secrets automatically matching the configuration in
litestream.yml.

```sh
fly storage create --app sheerluck-staging --name sheerluck-staging-backup
```

Now we are ready to deploy the app.

```sh
make cross-compile # required for the docker build.
fly deploy --app sheerluck-staging
```

## Attribution

Sheerluck logo made by Martin Yrjölä.

Images was created with the assistance of [DALL·E 2](https://openai.com/dall-e-2)
and [DALL·E 3](https://openai.com/dall-e-3).

HeroIcons made by [steveschoger](https://twitter.com/steveschoger). Available on https://heroicons.dev/.

Game Icons made by [Delapouite](https://delapouite.com/) and Skoll. Available on https://game-icons.net.
