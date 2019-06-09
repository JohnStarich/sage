FROM golang:1.12-alpine as builder

RUN apk add --no-cache \
            git \
            openssl

ENV CGO_ENABLED=0

WORKDIR /src
COPY . .
RUN go build -o /sage ./cmd/server/main.go

FROM scratch

COPY --from=builder /sage /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /data
ENTRYPOINT ["/sage"]
CMD ["/data/ledger.rules", "/data/ledger.journal", "/data/ofxclient.ini"]
VOLUME ["/data"]
