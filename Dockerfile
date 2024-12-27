# --------------------------------------------------------------------------------
#  Build stage for compiling the binary and preparing files for the scratch image.
# --------------------------------------------------------------------------------
FROM --platform=linux/amd64 golang:1.23.4-alpine3.21 AS build

ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64
ENV GOCACHE=/workspace/.cache

RUN apk add --no-cache \
    # Important: required for go-sqlite3
    gcc \
    # Required for Alpine
    musl-dev \
    # Update CA certificates
    ca-certificates \
    # Add time zone data
    tzdata \
    # Required for fetching and running Tailwind standalone
    curl

RUN adduser \
  --disabled-password \
  --gecos "" \
  --home "/nonexistent" \
  --shell "/sbin/nologin" \
  --no-create-home \
  --uid 65532 \
  sheerluck

WORKDIR /workspace/

COPY /go.mod .
COPY /go.sum .

RUN go mod download
RUN go mod verify

COPY /cmd ./cmd
COPY /internal ./internal

# Build a statically linked binary.
RUN go build -ldflags='-s -w -extldflags "-static"' -o /dist/sheerluck ./cmd/web

# Minimize CSS and copy UI files to dist
COPY /ui ./ui
RUN filehash=`md5sum ./ui/static/main.css | awk '{ print $1 }'` && \
    sed -i "s/\/main.css/\/main.${filehash}.css/g" ui/templates/base.gohtml && \
    mv ./ui/static/main.css ui/static/main.${filehash}.css
RUN cp -r ./ui /dist/ui

# -----------------------------------------------------------------------------
#  Dependency images including binaries we can copy over to the scratch image.
# -----------------------------------------------------------------------------
FROM --platform=linux/amd64 litestream/litestream:0.3.13 AS litestream
FROM --platform=linux/amd64 keinos/sqlite3:3.47.2 AS sqlite3

# -----------------------------------------------------------------------------
#  Main stage for copying files over to the scratch image.
# -----------------------------------------------------------------------------
FROM --platform=linux/amd64 scratch

COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group
COPY --from=build /dist /dist

# Configure Litestream for backups to object storage.
COPY /litestream.yml /etc/litestream.yml
COPY --from=litestream /usr/local/bin/litestream /dist/litestream

# Copy sqlite3 binary for database operations with `make fly-sqlite3` command.
COPY --from=sqlite3 /usr/bin/sqlite3 /usr/bin/sqlite3
COPY --from=sqlite3 /usr/lib/libz.so.1 /usr/lib/libz.so.1
COPY --from=sqlite3 /lib/ld-musl-x86_64.so.1 /lib/ld-musl-x86_64.so.1

USER sheerluck:sheerluck

ENV TZ=Europe/Helsinki
ENV SHEERLUCK_ADDR=":4000"
# pprof only available from internal network
ENV SHEERLUCK_PPROF_ADDR=":6060"
ENV SHEERLUCK_TEMPLATE_PATH="/dist/ui/templates"

EXPOSE 4000 6060 9090

WORKDIR /dist
ENTRYPOINT [ "./litestream", "replicate", "-exec", "./sheerluck" ]
