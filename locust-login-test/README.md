# 登录稳定性压测方案

## 目录结构

```
locust-login-test/
├── locustfile.py        # 网关层 HTTP 压测（token 校验）
├── grpc_bench.sh        # userservice gRPC 压测
├── proto/
│   └── user.proto       # proto 定义（根据你实际的修改）
├── testdata/
│   └── new_users.json   # gRPC 新用户测试数据
└── README.md
```

---

## 第一步：准备有效 token

手动登录一次，从浏览器 DevTools → Application → Cookies 里复制 `AccessCookieName` 的值，
填入 `locustfile.py` 顶部的 `VALID_TOKENS` 列表：

```python
VALID_TOKENS = [
    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",   # 从浏览器复制
    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",   # 多准备几个
]
```

> 建议准备 5～10 个 token，模拟不同用户 session 并发，避免全部共用同一个 token 被限流。

---

## 第二步：安装依赖

```bash
pip install locust

# macOS 安装 ghz
brew install ghz

# 或从官网下载：https://ghz.sh/docs/install
```

---

## 第三步：运行网关层压测（Locust）

```bash
cd locust-login-test
locust -f locustfile.py --host http://localhost:8080
```

打开浏览器访问 `http://localhost:8089`，配置：

| 参数 | 建议值 | 说明 |
|------|--------|------|
| Number of users | 50 | 从低并发开始 |
| Spawn rate | 5 | 每秒新增 5 个用户，阶梯加压 |
| Host | http://localhost:8080 | 已预填 |

观察指标：
- **Failure rate < 0.1%**（核心指标）
- **P95 < 200ms**（网关自身处理，不含 GitHub 调用）
- 压测结束后终端会打印**自定义登录报告**，重点看"有效token错误拒绝"和"无token错误放行"，两项必须为 0

---

## 第四步：运行 userservice gRPC 压测（ghz）

先确认 proto 文件字段与你实际实现一致，再运行：

```bash
chmod +x grpc_bench.sh
./grpc_bench.sh
```

脚本会依次跑三个场景：
1. 查询已存在用户（模拟 token 过期重新登录）
2. 批量创建新用户（模拟首次登录）
3. 50→100→150→200 并发阶梯加压，找到 userservice 的吞吐上限

结果 JSON 保存在 `/tmp/ghz_result_cXXX.json`，关注：
- **Requests/sec**：userservice 的实际 QPS 上限
- **p99**：尾延迟，目标 < 500ms
- **Error ratio**：错误率，目标 < 0.1%

---

## 常见问题

**Q: VALID_TOKENS 为空时 Locust 怎么跑？**
LoggedInUser 场景会自动跳过，只跑 AnonymousUser（无 token 拦截测试）。

**Q: ghz 报 "proto file not found"？**
确保在 `locust-login-test/` 目录下运行，或修改 `grpc_bench.sh` 中 `PROTO_PATH` 为绝对路径。

**Q: proto 字段对不上怎么办？**
根据你实际的 `LoginOrRegisterRequest` 字段修改 `proto/user.proto` 和 `grpc_bench.sh` 中的 `--data` 内容。
