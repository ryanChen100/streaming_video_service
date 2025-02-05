# 构建阶段
FROM golang:1.23-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git

# 设置工作目录
WORKDIR /app

# 假设 .yaml 文件位于 ./cmd/config 目录中
COPY ./cmd/api_gateway/config/api_gateway.yaml /app/config/api_gateway.yaml

# 缓存模块依赖
COPY go.mod go.sum ./
RUN go mod download

# 拷贝服务代码并构建
COPY . .
RUN go build -o api_gateway ./cmd/api_gateway

# 运行阶段
FROM alpine:latest

# 设置非 root 用户以提高安全性
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/api_gateway .

# 暴露容器端口（动态端口）
EXPOSE ${PORT}

# 定义构建参数


# 运行服务
CMD ["./api_gateway"]