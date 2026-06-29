# JetBrains HTTP Client 测试脚本

## 目录说明

- `00-system.http`：健康检查与指标接口
- `90-smoke.http`：CI smoke，仅跑健康检查
- `_shared/helpers.js`：通用断言与唯一值生成

## 环境文件

- `http-client.env.json`：公共环境变量（提交到仓库）
- `http-client.private.env.json`：本地私有变量（不提交，从 `.example` 复制）

## 使用方式

1. 复制 `http-client.private.env.json.example` 为 `http-client.private.env.json`
2. 填写 Bearer Token 等私有变量
3. 在 JetBrains IDE 中选择 `local` / `dev` / `test` 环境执行
