#build stage
FROM golang:1.24.3@sha256:4c0a1814a7c6c65ece28b3bfea14ee3cf83b5e80b81418453f0e9d5255a5d7b8 AS builder

RUN apt-get update && apt-get install -y \
    libzmq3-dev \
    pkg-config \
    git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY . .

RUN chmod +x ./scripts/build.sh

ARG CI=false
ARG VERSION=""
ARG COMMIT=""

RUN ./scripts/build.sh \
    $( [ "$CI" = "true" ] && echo "--ci" ) \
    $( [ -n "$VERSION" ] && echo "--version \"$VERSION\"" ) \
    $( [ -n "$COMMIT" ] && echo "--commit \"$COMMIT\"" )


#runtime stage
FROM debian:bookworm-slim@sha256:90522eeb7e5923ee2b871c639059537b30521272f10ca86fdbbbb2b75a8c40cd AS build-release-stage

RUN apt-get update && apt-get install -y \
    libzmq5 \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -u 1001 appuser && \
    mkdir -p /data && \
    mkdir -p /home/appuser/.local/share/cogmoteGO/experiments && \
    chown -R appuser:appuser /data /home/appuser/.local

ENV XDG_DATA_HOME=/home/appuser/.local/share

COPY --from=builder --chown=appuser:appuser /app/build/linux-amd64/cogmoteGO /usr/local/bin/

EXPOSE 9012

WORKDIR /data
VOLUME ["/data", "/home/appuser/.local/share"]

USER appuser

ENTRYPOINT ["cogmoteGO"]