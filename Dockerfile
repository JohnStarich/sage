FROM golang:1.12-alpine as builder

RUN apk add --no-cache \
            bash \
            git \
            make \
            npm \
            openssl

ENV CGO_ENABLED=0

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION
RUN make build


FROM chromedp/headless-shell:79.0.3945.45
ENV PATH=$PATH:/headless-shell

WORKDIR /data
ENTRYPOINT ["/sage"]
CMD ["-port=8080", "-rules=/data/ledger.rules", "-ledger=/data/ledger.journal", "-data=/data"]
VOLUME ["/data"]

COPY --from=builder /src/out/sage /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
