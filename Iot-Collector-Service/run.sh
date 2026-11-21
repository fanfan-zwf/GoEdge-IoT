#!/usr/bin/env bash
# Iot-Collector-Service 启动脚本
# 用法: ./run.sh
#   - 自动编译（若 bin/collector-service 不存在）
#   - 后台运行并写 PID 文件，配合 ./stop.sh 优雅停止

# 本脚本用到 bash 数组等特性；若被 sh/dash 调用（如 `sh run.sh`），自动改用 bash 重新执行。
# 这行必须放在所有 bash 专有语法之前，且本身要保持 POSIX 兼容。
if [ -z "${BASH_VERSION:-}" ]; then exec bash "$0" "$@"; fi

set -euo pipefail

# 以脚本所在目录为项目根：配置(configs/config.yaml)、日志(./logs)都是相对路径，
# 必须在项目根启动，否则读不到配置。
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

BIN_NAME="collector-service"
BIN_PATH="$SCRIPT_DIR/bin/$BIN_NAME"
PID_FILE="$SCRIPT_DIR/$BIN_NAME.pid"
LOG_DIR="$SCRIPT_DIR/logs"
# 每次启动生成独立的运行日志: run-[日期_时间].log，便于按时间排查崩溃/panic
RUN_LOG="$LOG_DIR/run-$(date +%Y-%m-%d_%H%M%S).log"
LIB_DIR="$SCRIPT_DIR/third_party/lib"

# 运行日志(run-*.log)最多保留几个（含本次），超出的最旧文件自动删除；设为 0 表示不清理
KEEP_RUN_LOGS=30

# snap7 动态库路径：CGO 的 rpath 是 $ORIGIN/third_party/lib，二进制在 bin/ 下时
# $ORIGIN 解析为 bin/，找不到库，故显式指向 third_party/lib。
export LD_LIBRARY_PATH="$LIB_DIR${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"

# 已在运行则不重复启动
if [ -f "$PID_FILE" ]; then
  OLD_PID="$(cat "$PID_FILE" 2>/dev/null || true)"
  if [ -n "${OLD_PID:-}" ] && kill -0 "$OLD_PID" 2>/dev/null; then
    echo "服务已在运行 (PID=$OLD_PID)，如需重启请先执行 ./stop.sh"
    exit 1
  fi
  # PID 文件残留但进程已不在，清理后继续启动
  rm -f "$PID_FILE"
fi

# 编译：仅当可执行文件缺失时构建，避免每次启动都重编
if [ ! -x "$BIN_PATH" ]; then
  echo "未找到可执行文件，开始编译: go build -o bin/$BIN_NAME ./cmd/server"
  go build -o "$BIN_PATH" ./cmd/server
else
  echo "使用已存在的可执行文件: $BIN_PATH（如需重新编译先执行 make clean 或删除该文件）"
fi

mkdir -p "$LOG_DIR"

echo "启动 $BIN_NAME ..."
# stdout/stderr 落到本次独立的运行日志（可捕获 logger 初始化前的早期 panic/崩溃堆栈）
nohup "$BIN_PATH" > "$RUN_LOG" 2>&1 &
PID=$!
echo "$PID" > "$PID_FILE"

# 清理旧的运行日志：文件名含时间戳，按名称排序即时间顺序，只保留最近 KEEP_RUN_LOGS 个（含本次）
if [ "$KEEP_RUN_LOGS" -gt 0 ]; then
  shopt -s nullglob
  logs=( "$LOG_DIR"/run-*.log )
  shopt -u nullglob
  # glob 结果按名称升序，下标 0 起为最旧，删除多出来的部分
  remove=$(( ${#logs[@]} - KEEP_RUN_LOGS ))
  for ((i = 0; i < remove; i++)); do
    rm -f "${logs[i]}"
  done
  if [ "$remove" -gt 0 ]; then
    echo "已清理 $remove 个旧运行日志（保留最近 $KEEP_RUN_LOGS 个）"
  fi
fi

# 简单健康检查：等待 2 秒确认进程存活（配置错误等会在此刻退出）
sleep 2
if kill -0 "$PID" 2>/dev/null; then
  echo "启动成功 (PID=$PID)"
  echo "  运行日志(本次): $RUN_LOG"
  echo "  业务日志目录:   $LOG_DIR"
  echo "  停止服务:       ./stop.sh"
else
  echo "启动失败，进程已退出，请查看运行日志: $RUN_LOG"
  rm -f "$PID_FILE"
  exit 1
fi
