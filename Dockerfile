FROM --platform=$BUILDPLATFORM golang:1.23 AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
RUN echo "I am running on $BUILDPLATFORM, building for $TARGETPLATFORM"

WORKDIR /app

# 安装必要的构建工具和库
RUN apt-get update && apt-get install -y \
    gcc-multilib \
    g++-multilib \
    gcc-mingw-w64 \
    g++-mingw-w64 \
    libc6-dev-i386 \
    gcc-aarch64-linux-gnu \
    g++-aarch64-linux-gnu \
    gcc-arm-linux-gnueabihf \
    g++-arm-linux-gnueabihf \
    libsqlite3-dev

COPY . .

# 设置交叉编译环境并构建
RUN case "$TARGETPLATFORM" in \
    "linux/amd64")  CC=gcc CGO_ENABLED=1 GOARCH=amd64 GOOS=linux BINARY_NAME=sixin-server_linux_amd64 ;; \
    "linux/386")    CC=gcc CGO_ENABLED=1 GOARCH=386 GOOS=linux BINARY_NAME=sixin-server_linux_386 ;; \
    "linux/arm64")  CC=aarch64-linux-gnu-gcc CGO_ENABLED=1 GOARCH=arm64 GOOS=linux BINARY_NAME=sixin-server_linux_arm64 ;; \
    "linux/arm/v7") CC=arm-linux-gnueabihf-gcc CGO_ENABLED=1 GOARCH=arm GOARM=7 GOOS=linux BINARY_NAME=sixin-server_linux_armv7 ;; \
    "linux/arm/v6") CC=arm-linux-gnueabihf-gcc CGO_ENABLED=1 GOARCH=arm GOARM=6 GOOS=linux BINARY_NAME=sixin-server_linux_armv6 ;; \
    "darwin/amd64") CC=o64-clang CGO_ENABLED=1 GOARCH=amd64 GOOS=darwin BINARY_NAME=sixin-server_darwin_amd64 ;; \
    "darwin/arm64") CC=o64-clang CGO_ENABLED=1 GOARCH=arm64 GOOS=darwin BINARY_NAME=sixin-server_darwin_arm64 ;; \
    "windows/amd64") CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOARCH=amd64 GOOS=windows BINARY_NAME=sixin-server_windows_amd64.exe ;; \
    "windows/386")   CC=i686-w64-mingw32-gcc CGO_ENABLED=1 GOARCH=386 GOOS=windows BINARY_NAME=sixin-server_windows_386.exe ;; \
    *) echo "Unsupported platform: $TARGETPLATFORM" && exit 1 ;; \
    esac \
    && export CC CGO_ENABLED GOARCH GOOS GOARM \
    && go build -v -tags 'sqlite_foreign_keys' -ldflags '-w -s' -o ${BINARY_NAME}

FROM scratch
COPY --from=builder /app/sixin-server* /
