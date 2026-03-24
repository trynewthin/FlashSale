# 本地启动与运行路径

## 1. 入口约定

当前仓库的命令入口以这两类为准：

- `go run ./ops/cmd ...`
- `bash ./ops.sh`

不再使用旧文档中提到的失效 CLI 入口。

## 2. 最小本地运行路径

### 启动基础环境

```bash
go run ./ops/cmd env up
```

### 执行迁移

```bash
go run ./ops/cmd env migrate-up
```

### 执行连通性检查

```bash
go run ./ops/cmd env smoke
```

## 3. 前端开发

三端前端均使用 Bun。

- `frontend/user`
- `frontend/admin`
- `frontend/ops`

常见动作包括安装依赖、运行开发服务器、构建和测试，但具体命令应在各自目录执行，避免把所有开发细节再堆回根 README。

## 4. 何时使用 `ops.sh`

`ops.sh` 适合承接面向维护者的高频操作入口。对于需要明确模块和参数的操作，优先直接使用 `go run ./ops/cmd`，这样更接近当前代码事实。
