FROM python:3-alpine as base

FROM base as builder

RUN apk add --no-cache \
        build-base \
        git \
        libffi-dev \
        libxml2-dev \
        openssl-dev \
        py3-cffi \
        py3-libxml2 \
        xmlsec-dev

COPY requirements.txt /
RUN pip3 install \
        --prefix=/install \
        -r /requirements.txt

FROM base

RUN apk add --no-cache \
        libffi \
        libxml2 \
        openssl \
        xmlsec

WORKDIR /src
ENTRYPOINT ["/src/sync.py"]
CMD ["--help"]

COPY --from=builder /install /usr/local

COPY . ./
