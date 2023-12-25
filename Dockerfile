FROM golang as base

RUN adduser \
  --disabled-password \
  --gecos "" \
  --home "/nonexistent" \
  --shell "/sbin/nologin" \
  --no-create-home \
  --uid 65532 \
  sheerluck

WORKDIR $GOPATH/src/sheerluck/

COPY /go.mod .
COPY /go.sum .

RUN go install golang.org/dl/go1.22rc1@latest
RUN go1.22rc1 download
RUN go1.22rc1 mod download
RUN go1.22rc1 mod verify

COPY /cmd ./cmd
COPY /internal ./internal
COPY /sqlite ./sqlite

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go1.22rc1 build -ldflags='-s -w' -trimpath -o /dist/sheerluck ./cmd/web
# Ensure all the dynamic libraries are available for the executable
RUN ldd /dist/sheerluck | tr -s [:blank:] '\n' | grep ^/ | xargs -I % install -D % /dist/%


FROM scratch

COPY --from=base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=base /etc/passwd /etc/passwd
COPY --from=base /etc/group /etc/group

COPY --from=base /dist /dist
COPY /ui /dist/ui
COPY /.env /dist

USER sheerluck:sheerluck

CMD ["/dist/sheerluck"]
