# 单词拼写挑战

一个自包含的 Go 1.22.12 单模块应用。服务端提供固定的简单/中等词库、答题计分、用时记录、错词回放、历史统计和团队验收汇总；前端是随 Go 二进制嵌入的可操作验收界面。

## 运行

```bash
go run ./cmd/spelling-challenge
```

浏览器访问 `http://127.0.0.1:8080`。也可以用 `-addr` 指定监听地址：

```bash
go run ./cmd/spelling-challenge -addr 0.0.0.0:8080
```

应用不依赖数据库、网络服务、随机数或服务端时钟。词库、验收记录均为进程内固定 fixture，重启后恢复初始状态。

## 前端构建

前端要求 Node.js 20，无第三方 npm 依赖：

```bash
cd web
npm ci
npm run build
```

构建输出位于 `web/dist/`，运行 Go 服务不需要预先构建该目录。

## 测试

```bash
CGO_ENABLED=0 go test -count=1 ./...
```

测试覆盖两个词库、正确和错误答题、分数、用时、错词、历史统计、顺序验收及并发验收。并发验收用固定同步点让两名操作员同时更新同一记录；当前版本按任务要求保留丢失更新缺陷，因此对应业务链路断言会稳定失败。

## 接口

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/words?level=simple` | 获取指定难度词库 |
| `POST` | `/api/attempts` | 保存答题结果 |
| `GET` | `/api/attempts` | 查看答题历史 |
| `GET` | `/api/stats` | 查看汇总统计 |
| `GET` | `/api/mistakes` | 查看错词回放 |
| `GET` | `/api/reviews/daily-001` | 查看验收汇总 |
| `POST` | `/api/reviews/daily-001/confirm` | 提交操作员确认 |
