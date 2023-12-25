# -----------------------------------------------------------------------------
#  Build Stage
# -----------------------------------------------------------------------------
FROM golang:alpine3.18 AS build

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
    tzdata

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

RUN go install golang.org/dl/go1.22rc1@latest
RUN go1.22rc1 download
RUN go1.22rc1 mod download
RUN go1.22rc1 mod verify

COPY /cmd ./cmd
COPY /internal ./internal
COPY /sqlite ./sqlite

# Build a statically linked binary
RUN go1.22rc1 build -ldflags='-s -w -extldflags "-static"' -o /dist/sheerluck ./cmd/web

# -----------------------------------------------------------------------------
#  Main Stage
# -----------------------------------------------------------------------------
FROM scratch

COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group
COPY --from=build /dist /dist
COPY /ui /dist/ui
COPY /.env /dist
COPY /sqlite/init.sql /dist/sqlite/init.sql

USER sheerluck:sheerluck

ENV TZ=Europe/Helsinki

EXPOSE 4000

WORKDIR /dist
ENTRYPOINT [ "./sheerluck", "-addr", ":4000" ]