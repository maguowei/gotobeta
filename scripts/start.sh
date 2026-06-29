#!/bin/sh
# 本地开发脚本：用 `go run` 启动 server，便于热加载和快速迭代。
# 容器/生产请直接执行 Docker CMD ["/app/server"]；K8s deployment 不要把此脚本作为 image entrypoint。
set -e
APP_ENV=${APP_ENV:-local}
export APP_ENV
exec go run ./cmd/server "$@"
