# 构建阶段
FROM golang:1.23-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git

# 设置工作目录
WORKDIR /app

# 假设 .yaml 文件位于 ./cmd/config 目录中
COPY ./cmd/streaming_service/config/streaming_service.yaml /app/config/streaming_service.yaml

# 缓存模块依赖
COPY go.mod go.sum ./
RUN go mod download

# 拷贝服务代码并构建
COPY . .
RUN go build -o streaming_service ./cmd/streaming_service

# 运行阶段
FROM alpine:latest

# 安装 ffmpeg（以及其它可能需要的运行时依赖）
RUN apk add --no-cache ffmpeg

# 设置非 root 用户以提高安全性
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/streaming_service .

# 暴露容器端口（动态端口）
EXPOSE ${PORT}


# 运行服务
CMD ["./streaming_service"]
