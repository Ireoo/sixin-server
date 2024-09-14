FROM --platform=$BUILDPLATFORM golang:1.23 AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
RUN echo "I am running on $BUILDPLATFORM, building for $TARGETPLATFORM"

WORKDIR /app

# # 更新包列表并安装必要的构建工具和库
# RUN apt-get update && apt-get install -y --no-install-recommends \
#     gcc \
#     libc6-dev \
#     libsqlite3-dev \
#     && rm -rf /var/lib/apt/lists/*

# # 对于交叉编译，我们可能需要额外的编译器
# RUN if [ "$BUILDPLATFORM" != "$TARGETPLATFORM" ]; then \
#     apt-get update && apt-get install -y --no-install-recommends \
#     gcc-aarch64-linux-gnu \
#     gcc-arm-linux-gnueabihf \
#     && rm -rf /var/lib/apt/lists/*; \
#     fi

COPY . .

# 设置交叉编译环境并构建
RUN case "$TARGETPLATFORM" in \
    "linux/amd64")  CC=gcc CGO_ENABLED=1 GOARCH=amd64 GOOS=linux BINARY_NAME=sixin-server_linux_amd64 ;; \
    "linux/386")    CC=gcc CGO_ENABLED=1 GOARCH=386 GOOS=linux BINARY_NAME=sixin-server_linux_386 ;; \
    "linux/arm64")  CC=aarch64-linux-gnu-gcc CGO_ENABLED=1 GOARCH=arm64 GOOS=linux BINARY_NAME=sixin-server_linux_arm64 ;; \
    "linux/arm/v7") CC=arm-linux-gnueabihf-gcc CGO_ENABLED=1 GOARCH=arm GOARM=7 GOOS=linux BINARY_NAME=sixin-server_linux_armv7 ;; \
    "linux/arm/v6") CC=arm-linux-gnueabihf-gcc CGO_ENABLED=1 GOARCH=arm GOARM=6 GOOS=linux BINARY_NAME=sixin-server_linux_armv6 ;; \
    "windows/amd64") GOOS=windows GOARCH=amd64 CGO_ENABLED=1 BINARY_NAME=sixin-server_windows_amd64.exe ;; \
    "windows/386")   GOOS=windows GOARCH=386 CGO_ENABLED=1 BINARY_NAME=sixin-server_windows_386.exe ;; \
    "darwin/amd64")  CC=gcc CGO_ENABLED=1 GOARCH=amd64 GOOS=darwin BINARY_NAME=sixin-server_darwin_amd64 ;; \
    "darwin/arm64")  CC=gcc CGO_ENABLED=1 GOARCH=arm64 GOOS=darwin BINARY_NAME=sixin-server_darwin_arm64 ;; \
    *) echo "Unsupported platform: $TARGETPLATFORM" && exit 1 ;; \
    esac \
    && CGO_ENABLED=1 go build -v -o {BINARY_NAME}

FROM scratch
COPY --from=builder /app/sixin-server* /
