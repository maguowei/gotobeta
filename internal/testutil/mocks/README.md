# mocks

存放 `mockgen` 生成的 mock 实现。

## 生成方式

```bash
mockgen -source=internal/modules/<bc>/domain/repository/repository.go \
        -destination=internal/testutil/mocks/mock_<bc>_repo.go \
        -package=mocks
```

## 命名规范

- 文件名：`mock_<entity>_<type>.go`（如 `mock_todo_repo.go`、`mock_todo_service.go`）
- 包名统一为 `mocks`
