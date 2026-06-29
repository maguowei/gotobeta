# 发布记录

按版本记录每次发布的变更内容、升级注意事项和回滚方案。

## 发布前检查

```bash
env GOCACHE=/tmp/go-build-cache GOMODCACHE=/tmp/go-mod-cache make verify
make smoke
make lefthook-run
make lint-secrets
make vuln-check
```

涉及运行时资产时补充：

```bash
docker compose config
kubectl apply --dry-run=client -f deployments/kubernetes/
```

如果工具缺失，记录缺失项和已运行的最强子集。不要把未执行的验证写成已通过。

## 记录格式

```
## vX.Y.Z (YYYY-MM-DD)

### 新增
- ...

### 变更
- ...

### 修复
- ...

### 升级注意
- ...

### 验证
- `make verify`: ...
- `make smoke`: ...
- `docker compose config`: ...
- `kubectl apply --dry-run=client -f deployments/kubernetes/`: ...

### 回滚
- 回滚版本：
- 数据回滚：
- 配置回滚：
- 验证方式：
```

## 记录原则

- 每次发布都写清楚影响面：API、数据库、配置、事件、观测、部署、运维脚本。
- 数据库或事件 schema 变化必须说明兼容性和回滚策略。
- 配置项变化必须说明默认值、环境变量覆盖方式和生产注意事项。
- 不写真实 token、真实 DSN、真实 registry、真实 Kafka topic 或内部域名。
