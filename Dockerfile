# -----------------------------------------------------------------------------
#  Build Stage
# -----------------------------------------------------------------------------
FROM --platform=linux/amd64 golang:1.23.4-alpine3.21 AS build

ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64

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

WORKDIR /github.com/myrjola/sheerluck/

COPY /go.mod .
COPY /go.sum .

RUN go mod download
RUN go mod verify

COPY /cmd ./cmd
COPY /internal ./internal

# Build a statically linked binary
RUN go build -ldflags='-s -w -extldflags "-static"' -o /dist/sheerluck ./cmd/web

# Minimize CSS and copy UI files to dist
COPY /ui ./ui
RUN filehash=`md5sum ./ui/static/main.css | awk '{ print $1 }'` && \
    sed -i "s/\/main.css/\/main.${filehash}.css/g" ui/templates/base.gohtml && \
    mv ./ui/static/main.css ui/static/main.${filehash}.css
RUN cp -r ./ui /dist/ui

# -----------------------------------------------------------------------------
#  Main Stage
# -----------------------------------------------------------------------------
FROM --platform=linux/amd64 scratch

COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group
COPY --from=build /dist /dist
COPY /.env /dist

# Configure Litestream for backups to object storage
COPY /litestream.yml /etc/litestream.yml
COPY --from=litestream/litestream:0.3.13 /usr/local/bin/litestream /dist/litestream

USER sheerluck:sheerluck

ENV TZ=Europe/Helsinki

EXPOSE 4000 6060

WORKDIR /dist
ENTRYPOINT [ "./litestream", "replicate", "-exec", "./sheerluck -addr :4000 -pprof-addr :6060" ]
