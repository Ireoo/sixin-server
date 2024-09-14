FROM --platform=$BUILDPLATFORM golang:1.23 AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
RUN echo "I am running on $BUILDPLATFORM, building for $TARGETPLATFORM"

WORKDIR /app

# 更新包列表并安装必要的构建工具和库
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

COPY . .

# 设置交叉编译环境并构建
RUN BINARY_NAME=$(echo "sixin-server_${TARGETPLATFORM}" | tr '/' '_') \
    && if [[ "$TARGETPLATFORM" == windows* ]]; then \
    BINARY_NAME="${BINARY_NAME}.exe"; \
    fi \
    && CGO_ENABLED=1 go build -v -o "$BINARY_NAME"

RUN ls -lrt

FROM scratch
COPY --from=builder /app/sixin-server_* /
