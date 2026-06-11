FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY go.mod ./
COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /iplist-proxy .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /iplist-proxy /app/iplist-proxy

ENV LISTEN_ADDR=:8090
ENV UPSTREAM_URL=https://iplist.opencck.org/

EXPOSE 8090

ENTRYPOINT ["/app/iplist-proxy"]
