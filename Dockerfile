FROM python:3-alpine as base

FROM base as deps

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

COPY --from=deps /install /usr/local

WORKDIR /data
ENTRYPOINT ["/src/sync.py"]
CMD ["--server"]
VOLUME ["/data"]

RUN ln -s /data/ofxclient.ini ~/ofxclient.ini

# Use the simple keyring to simplify where passwords are stored.
# TODO move to an encrypted-at-rest keyring
RUN path=~/.local/share/python_keyring; \
        mkdir -p "$path" && \
        echo $'\
[backend]\n\
default-keyring=simplekeyring.SimpleKeyring\n\
' > "$path"/keyringrc.cfg

ENV LEDGER_FILE=/data/ledger.journal
ENV LEDGER_RULES_FILE=/data/ledger.rules
ENV SYNC_EMBEDDED=true

ENV PYTHONPATH=/src
COPY . /src
