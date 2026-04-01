#!/usr/bin/env bash
# ────────────────────────────────────────────────────────────
# userservice gRPC 压测脚本（使用 ghz）
#
# 安装 ghz：
#   brew install ghz          # macOS
#   或从 https://ghz.sh 下载二进制
#
# 使用方式：
#   chmod +x grpc_bench.sh
#   ./grpc_bench.sh
# ────────────────────────────────────────────────────────────

USERSERVICE_ADDR="localhost:50051"
PROTO_PATH="./proto/user.proto"
METHOD="user.UserService/LoginOrRegister"

# 结果保存到项目目录，方便留存对比
mkdir -p ./results

# ── 场景一：查询已存在用户（模拟 token 过期后重新登录）──────
echo "▶ 场景一：正常负载 - 查询已存在用户"
ghz \
  --proto   "$PROTO_PATH" \
  --call    "$METHOD" \
  --insecure \
  --concurrency 100 \
  --total       1000 \
  --timeout     5s \
  --data '{
    "provider": "github",
    "open_id":  "147716474",
    "nickname": "KuoZ",
    "avatar":   "https://avatars.githubusercontent.com/u/147716474?v=4",
    "email":    "2241045762@qq.com"
  }' \
  "$USERSERVICE_ADDR"

echo ""
echo "────────────────────────────────────────"

# ── 场景三：阶梯加压，找到吞吐上限 ───────────────────────
echo "▶ 场景三：阶梯加压（50 → 200 并发）"
for CONCURRENCY in 50 100 150 200; do
  echo "  并发数: $CONCURRENCY"
  ghz \
    --proto   "$PROTO_PATH" \
    --call    "$METHOD" \
    --insecure \
    --concurrency "$CONCURRENCY" \
    --total        500 \
    --duration     30s \
    --timeout      5s \
    --data '{
      "provider": "github",
      "open_id":  "147716474",
      "nickname": "KuoZ",
      "avatar":   "https://avatars.githubusercontent.com/u/147716474?v=4",
      "email":    "2241045762@qq.com"
    }' \
    --output "./results/ghz_result_c${CONCURRENCY}.json" \
    --format json \
    "$USERSERVICE_ADDR" \
  | grep -E "Summary|Requests/sec|Fastest|Slowest|Average|p95|p99|Error"
  echo ""
done

echo "✅ 压测完成，JSON 报告保存在 ./results/ 目录"