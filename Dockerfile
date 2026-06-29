# 构建阶段
FROM golang:1.26.3-alpine3.23 AS builder

WORKDIR /app

ENV CGO_ENABLED=0
ENV GOPROXY=https://proxy.golang.org,direct

# 安装系统依赖
RUN apk add --no-cache git

# 复制 go mod 文件
COPY go.mod go.sum ./

RUN go mod download

# 复制源代码
COPY . .

# 生成 Ent 代码
RUN go generate ./internal/ent

# 编译
RUN go build -o /bin/server ./cmd/server
RUN go build -o /bin/datainit ./cmd/datainit
RUN go build -o /bin/migrate ./cmd/migrate

# 运行阶段
FROM alpine:3.23.3

RUN apk --no-cache add ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    addgroup -S -g 10001 app && \
    adduser -S -D -H -u 10001 -G app app

WORKDIR /app

# 复制二进制文件
COPY --from=builder --chown=app:app /bin/server /app/server
COPY --from=builder --chown=app:app /bin/datainit /app/datainit
COPY --from=builder --chown=app:app /bin/migrate /app/migrate

# 复制配置文件
COPY --chown=app:app configs /app/configs

USER app:app

EXPOSE 8080

CMD ["/app/server"]
