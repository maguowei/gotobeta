.PHONY: help run run-dev run-test migrate test test-architecture coverage-collect test-coverage coverage-check build build-linux generate fmt tidy lint lint-actions lint-go lint-openapi lint-secrets modernize-check mod-verify vuln-check smoke verify pre-push-verify clean tools-download lefthook-install lefthook-validate lefthook-run docker-build docker-run docker-stop docker-clean docker-logs docker-shell docker-push test-integration-compile test-integration

DOCKER_IMAGE ?= example.com/codego/gotobeta
GIT_BRANCH   := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null | tr '/' '-')
GIT_COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
DOCKER_TAG   ?= $(GIT_BRANCH)-$(GIT_COMMIT)
COVERAGE_THRESHOLD ?= 70
GOLANGCI_LINT_CACHE ?= $(CURDIR)/.cache/golangci-lint
GO_TOOL_GOCACHE ?= $(CURDIR)/.cache/go-build
GO_TOOL_GOMODCACHE ?= $(CURDIR)/.cache/go-mod
ACTIONLINT ?= go tool actionlint
GOVULNCHECK ?= go tool govulncheck
LEFTHOOK ?= go tool lefthook
VACUUM ?= go tool vacuum
GITLEAKS ?= go tool gitleaks
GOLANGCI_LINT ?= go tool golangci-lint
GO_TOOL_MODULES := github.com/daveshanley/vacuum github.com/evilmartians/lefthook/v2 github.com/golangci/golangci-lint/v2 github.com/rhysd/actionlint github.com/zricethezav/gitleaks/v8 golang.org/x/vuln

help: ## 显示帮助信息
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

run: ## 运行本地服务
	APP_ENV=local go run ./cmd/server

run-dev: ## 使用 dev 配置运行服务
	APP_ENV=dev go run ./cmd/server

run-test: ## 使用 test 配置运行服务
	APP_ENV=test go run ./cmd/server

migrate: ## 运行数据库迁移
	APP_ENV=local go run ./cmd/migrate

test: ## 运行测试
	go test $(shell go list ./... | grep -v ./internal/integration)

test-architecture: ## 校验 DDD 分层依赖边界
	go test ./internal/architecture -count=1

coverage-collect: ## 收集并过滤覆盖率数据
# 成功时静默（吞掉每个包的 coverage 行），失败时回显完整测试日志并非零退出，
# 保证测试失败仍可见且不破坏 make verify 的失败语义。
	@go test -coverprofile=coverage.out $(shell go list ./... | grep -v ./internal/integration) > coverage.test.log 2>&1 \
		|| { cat coverage.test.log; rm -f coverage.test.log; exit 1; }
	@rm -f coverage.test.log
# 过滤掉两类不计入单测覆盖率的代码：ent 生成代码，以及进程入口/组合根装配代码
# （cmd/* 主函数、server.RunHTTP/RunMigrate、datainit.Run、worker.Run）。
# 这些只做依赖装配并接入真实 DB/Kafka，由 make smoke 编译验证、由集成测试覆盖运行时，
# 与已排除的 ent / internal/integration 同理，避免装配代码稀释业务逻辑的真实覆盖率。
	@{ head -n 1 coverage.out; tail -n +2 coverage.out | grep -v '/internal/ent/' | grep -vE '/internal/app/server/|/internal/app/datainit/|/internal/app/worker/server\.go|/cmd/' || true; } > coverage.filtered.out

test-coverage: coverage-collect ## 运行测试并生成覆盖率报告
	go tool cover -html=coverage.filtered.out -o coverage.html
	go tool cover -func=coverage.filtered.out | grep total

coverage-check: coverage-collect ## 检查测试覆盖率是否达到基线
	@COVERAGE=$$(go tool cover -func=coverage.filtered.out | awk '/^total:/ {gsub("%", "", $$3); print $$3}'); \
	echo "Total coverage: $$COVERAGE%"; \
	awk -v coverage="$$COVERAGE" -v threshold="$(COVERAGE_THRESHOLD)" 'BEGIN { exit !(coverage + 0 >= threshold + 0) }' || { \
		echo "coverage $$COVERAGE% is below $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	}

build: ## 构建当前平台二进制
	mkdir -p bin
	go build -o bin/server ./cmd/server
	go build -o bin/migrate ./cmd/migrate
	go build -o bin/datainit ./cmd/datainit

build-linux: ## 构建 Linux x64 二进制
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/server-linux-amd64 ./cmd/server
	GOOS=linux GOARCH=amd64 go build -o bin/migrate-linux-amd64 ./cmd/migrate
	GOOS=linux GOARCH=amd64 go build -o bin/datainit-linux-amd64 ./cmd/datainit

generate: ## 生成代码
	go mod download entgo.io/ent
	go generate ./internal/ent

fmt: ## 格式化 Go 代码
	go fmt ./...

tidy: ## 整理依赖
	go mod tidy

lint: lint-go lint-openapi ## 运行代码与 OpenAPI 检查

tools-download: ## 下载 Go tool 依赖
	GOMODCACHE=$(GO_TOOL_GOMODCACHE) go mod download $(GO_TOOL_MODULES)

modernize-check: ## 检查 go fix 现代化迁移建议
	@diff=$$(GOCACHE=$(GO_TOOL_GOCACHE) go fix -diff ./... 2>/dev/null) || true; \
	if [ -n "$$diff" ]; then printf '%s\n' "$$diff"; exit 1; fi

lint-actions: tools-download ## 校验 GitHub Actions workflow
	GOCACHE=$(GO_TOOL_GOCACHE) GOMODCACHE=$(GO_TOOL_GOMODCACHE) $(ACTIONLINT) .github/workflows/*.yml

lint-go: tools-download ## 运行 golangci-lint
	mkdir -p $(GOLANGCI_LINT_CACHE)
	# --allow-parallel-runners: 允许 make smoke 并发跑多个生成项目的 lint 时不拿全局文件锁
	GOCACHE=$(GO_TOOL_GOCACHE) GOMODCACHE=$(GO_TOOL_GOMODCACHE) GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) $(GOLANGCI_LINT) run --allow-parallel-runners ./...

lint-openapi: tools-download ## 校验 OpenAPI 契约
	GOCACHE=$(GO_TOOL_GOCACHE) GOMODCACHE=$(GO_TOOL_GOMODCACHE) $(VACUUM) lint -d --no-banner --no-style --fail-severity warn -r vacuum-ruleset.yaml api/openapi.yaml

lint-secrets: tools-download ## 扫描硬编码密钥
	GOCACHE=$(GO_TOOL_GOCACHE) GOMODCACHE=$(GO_TOOL_GOMODCACHE) $(GITLEAKS) dir --redact --no-banner .

mod-verify: ## 校验模块缓存完整性
	go mod verify

vuln-check: ## 扫描 Go 已知漏洞
	GOCACHE=$(GO_TOOL_GOCACHE) $(GOVULNCHECK) ./...

smoke: ## 运行冒烟验证
	$(MAKE) generate
	$(MAKE) tidy
	$(MAKE) coverage-check
	$(MAKE) test-architecture
	$(MAKE) test-integration-compile
	$(MAKE) build

verify: ## 运行完整质量门禁
	@echo "==> [1/12] generate 代码生成"
	@$(MAKE) --no-print-directory -s generate
	@echo "✅ [1/12] generate 代码生成"
	@echo "==> [2/12] tidy 依赖整理"
	@$(MAKE) --no-print-directory -s tidy
	@echo "✅ [2/12] tidy 依赖整理"
	@echo "==> [3/12] modernize-check 现代化检查"
	@$(MAKE) --no-print-directory -s modernize-check
	@echo "✅ [3/12] modernize-check 现代化检查"
	@echo "==> [4/12] lint-actions GitHub Actions 校验"
	@$(MAKE) --no-print-directory -s lint-actions
	@echo "✅ [4/12] lint-actions GitHub Actions 校验"
	@echo "==> [5/12] lint 代码与 OpenAPI 检查"
	@$(MAKE) --no-print-directory -s lint
	@echo "✅ [5/12] lint 代码与 OpenAPI 检查"
	@echo "==> [6/12] lint-secrets 密钥扫描"
	@$(MAKE) --no-print-directory -s lint-secrets
	@echo "✅ [6/12] lint-secrets 密钥扫描"
	@echo "==> [7/12] mod-verify 模块完整性校验"
	@$(MAKE) --no-print-directory -s mod-verify
	@echo "✅ [7/12] mod-verify 模块完整性校验"
	@echo "==> [8/12] vuln-check 依赖漏洞扫描"
	@$(MAKE) --no-print-directory -s vuln-check
	@echo "✅ [8/12] vuln-check 依赖漏洞扫描"
	@echo "==> [9/12] coverage-check 测试与覆盖率"
	@$(MAKE) --no-print-directory -s coverage-check
	@echo "✅ [9/12] coverage-check 测试与覆盖率"
	@echo "==> [10/12] test-architecture 分层依赖校验"
	@$(MAKE) --no-print-directory -s test-architecture
	@echo "✅ [10/12] test-architecture 分层依赖校验"
	@echo "==> [11/12] test-integration-compile 集成测试编译"
	@$(MAKE) --no-print-directory -s test-integration-compile
	@echo "✅ [11/12] test-integration-compile 集成测试编译"
	@echo "==> [12/12] build 构建二进制"
	@$(MAKE) --no-print-directory -s build
	@echo "✅ [12/12] build 构建二进制"
	@echo "✅ verify 完成：全部检查通过"

pre-push-verify: ## 运行带本地成功缓存的 pre-push 门禁
	bash .lefthook/pre-push/verify-cache.sh

clean: ## 清理构建产物
	rm -rf bin/
	rm -f coverage.out coverage.filtered.out coverage.html coverage.test.log

lefthook-install: tools-download ## 安装 Lefthook Git hooks
	GOCACHE=$(GO_TOOL_GOCACHE) GOMODCACHE=$(GO_TOOL_GOMODCACHE) $(LEFTHOOK) install

lefthook-validate: tools-download ## 校验 Lefthook 配置
	GOCACHE=$(GO_TOOL_GOCACHE) GOMODCACHE=$(GO_TOOL_GOMODCACHE) $(LEFTHOOK) validate

lefthook-run: tools-download ## 运行 Lefthook pre-commit hooks
	GOCACHE=$(GO_TOOL_GOCACHE) GOMODCACHE=$(GO_TOOL_GOMODCACHE) $(LEFTHOOK) run pre-commit --all-files

docker-build: tidy ## 构建 Docker 镜像
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: ## 运行 Docker 容器
	docker run -d --name gotobeta -p 8080:8080 -v $(PWD)/configs:/app/configs $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-stop: ## 停止并删除 Docker 容器
	-docker stop gotobeta
	-docker rm gotobeta

docker-clean: ## 清理 Docker 容器和镜像
	-docker stop gotobeta
	-docker rm gotobeta
	-docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-logs: ## 查看 Docker 容器日志
	docker logs -f gotobeta

docker-shell: ## 进入 Docker 容器
	docker exec -it gotobeta sh

docker-push: ## 推送 Docker 镜像
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

test-integration-compile: ## 只编译 integration build tag，不启动 Docker 容器
	go test -tags integration -run '^$$' ./internal/integration/... -count=1

test-integration: ## 运行集成测试（需要本地 Docker 运行中）
	go test -tags integration ./internal/integration/... -count=1
