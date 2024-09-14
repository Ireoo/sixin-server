FROM --platform=$BUILDPLATFORM golang:1.23 AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
RUN echo "I am running on $BUILDPLATFORM, building for $TARGETPLATFORM"

WORKDIR /app

COPY . .

RUN apt-get update && apt-get install -y gcc-multilib g++-multilib

RUN case "$TARGETPLATFORM" in \
    "linux/amd64")  GOARCH=amd64  ;; \
    "linux/386")    GOARCH=386    ;; \
    "linux/arm64")  GOARCH=arm64  ;; \
    "linux/arm/v7") GOARCH=arm GOARM=7 ;; \
    "linux/arm/v6") GOARCH=arm GOARM=6 ;; \
    "darwin/amd64") GOARCH=amd64 GOOS=darwin ;; \
    "darwin/arm64") GOARCH=arm64 GOOS=darwin ;; \
    "windows/amd64") GOARCH=amd64 GOOS=windows ;; \
    "windows/386")   GOARCH=386   GOOS=windows ;; \
    *) echo "Unsupported platform: $TARGETPLATFORM" && exit 1 ;; \
    esac \
    && export GOARCH GOOS GOARM \
    && go build -v -o sixin-server

FROM scratch
COPY --from=builder /app/sixin-server /sixin-server
