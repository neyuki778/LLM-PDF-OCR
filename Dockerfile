# ---- Build Stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /build

# 先单独复制依赖文件，利用 Docker layer 缓存
COPY go.mod go.sum ./
RUN go mod download

# 复制全部源码并编译
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o server \
        ./cmd/server

# ---- Runtime Stage ----
FROM alpine:3.21

# ca-certificates: 对外 HTTPS 请求（Gemini / MinerU API）必需
# tzdata: 时区支持
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/server ./server
# 静态前端资源（index.html / auth.html / style.css / dist/）
COPY web/ ./web/

# output/ 和 data/ 在运行时通过 volume 挂载，不打进镜像

EXPOSE 8080
ENTRYPOINT ["./server"]
