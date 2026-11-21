#!/usr/bin/env bash
# Iot-Collector-Service 停止脚本（优雅退出）
# 用法: ./stop.sh
#   - 向 run.sh 记录的进程发送 SIGTERM，触发 main.go 里的优雅收尾
#   - 超过 GRACE_TIMEOUT 仍未退出则 SIGKILL 强制结束

# 若被 sh/dash 调用（如 `sh stop.sh`），自动改用 bash 重新执行（${BASH_SOURCE}/pipefail 为 bash 特性）。
if [ -z "${BASH_VERSION:-}" ]; then exec bash "$0" "$@"; fi

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

BIN_NAME="collector-service"
PID_FILE="$SCRIPT_DIR/$BIN_NAME.pid"
# 服务内部优雅退出预算为 15s，这里留冗余再等一等才强制
GRACE_TIMEOUT=20

if [ ! -f "$PID_FILE" ]; then
  echo "未找到 PID 文件 ($PID_FILE)，服务可能未通过 ./run.sh 启动"
  echo "可手动查找: pgrep -f 'bin/$BIN_NAME'"
  exit 1
fi

PID="$(cat "$PID_FILE" 2>/dev/null || true)"
if [ -z "${PID:-}" ] || ! kill -0 "$PID" 2>/dev/null; then
  echo "PID=$PID 对应的进程不存在，清理失效的 PID 文件"
  rm -f "$PID_FILE"
  exit 0
fi

echo "向 $BIN_NAME (PID=$PID) 发送 SIGTERM，优雅停止..."
kill -TERM "$PID"

# 轮询等待进程优雅退出
for ((i = 1; i <= GRACE_TIMEOUT; i++)); do
  if ! kill -0 "$PID" 2>/dev/null; then
    echo "已优雅停止 (用时 ${i}s)"
    rm -f "$PID_FILE"
    exit 0
  fi
  sleep 1
done

echo "优雅退出超时 (${GRACE_TIMEOUT}s)，发送 SIGKILL 强制结束..."
kill -KILL "$PID" 2>/dev/null || true
sleep 1
if kill -0 "$PID" 2>/dev/null; then
  echo "强制结束失败，请手动检查进程 PID=$PID"
  exit 1
fi
rm -f "$PID_FILE"
echo "已强制停止"
