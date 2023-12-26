# -----------------------------------------------------------------------------
#  Build Stage
# -----------------------------------------------------------------------------
FROM golang:1.22rc1-alpine3.19 AS build

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

WORKDIR /workspace

COPY /go.mod .
COPY /go.sum .

RUN go mod download
RUN go mod verify

COPY /cmd ./cmd
COPY /internal ./internal
COPY /sqlite ./sqlite

# Build a statically linked binary
RUN go build -ldflags='-s -w -extldflags "-static"' -o /dist/sheerluck ./cmd/web

# Minimize CSS and copy UI files to dist
COPY /ui ./ui
COPY /tailwind.config.js ./tailwind.config.js
COPY /input.css ./input.css
RUN curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/download/v3.4.0/tailwindcss-linux-x64 && \
  chmod +x tailwindcss-linux-x64 && \
  mv tailwindcss-linux-x64 tailwindcss
RUN ./tailwindcss --input ./input.css --output ./ui/static/main.css --minify
RUN filehash=`md5sum ./ui/static/main.css | awk '{ print $1 }'` && \
    sed -i "s/\/main.css/\/main.${filehash}.css/g" ui/html/base.gohtml && \
    mv ./ui/static/main.css ui/static/main.${filehash}.css
RUN cp -r ./ui /dist/ui

# -----------------------------------------------------------------------------
#  Main Stage
# -----------------------------------------------------------------------------
FROM scratch

COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group
COPY --from=build /dist /dist
COPY /.env /dist
COPY /sqlite/init.sql /dist/sqlite/init.sql

# Configure Litestream for backups to object storage
COPY /litestream.yml /etc/litestream.yml
COPY --from=litestream/litestream:0.3.13 /usr/local/bin/litestream /dist/litestream

USER sheerluck:sheerluck

ENV TZ=Europe/Helsinki

EXPOSE 4000 6060

WORKDIR /dist
ENTRYPOINT [ "./litestream", "replicate", "-exec", "./sheerluck -addr :4000 -pprof-addr :6060" ]