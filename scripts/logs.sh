#!/bin/bash
set -e

# GIVErS ログ表示ユーティリティ
#
# 使い方:
#   ./scripts/logs.sh                                    # 全サービスのログを follow
#   ./scripts/logs.sh backend                            # backend のログを follow
#   ./scripts/logs.sh nginx                              # nginx アクセスログを tail
#   ./scripts/logs.sh nginx error                        # nginx エラーログを tail
#   ./scripts/logs.sh --tail 100 backend                 # 最新100行を表示
#   ./scripts/logs.sh --remote backend                   # リモートサーバーのログ
#   ./scripts/logs.sh --since "1h" backend               # 直近1時間のログ
#   ./scripts/logs.sh --since "2026-03-01" --until "2026-03-02" backend
#   ./scripts/logs.sh --since "1h" backend -o out.log    # ファイルに出力

REMOTE_HOST="givers-conoha-root"
REMOTE_DIR="/opt/givers"
COMPOSE_FILE="docker-compose.prod.yml"
TAIL_LINES="50"
REMOTE=false
SERVICE=""
FOLLOW=true
SINCE=""
UNTIL=""
OUTPUT_FILE=""

# --- 引数解析 ---
while [[ $# -gt 0 ]]; do
  case "$1" in
    --remote|-r)   REMOTE=true; shift ;;
    --tail|-n)     TAIL_LINES="$2"; shift 2 ;;
    --no-follow)   FOLLOW=false; shift ;;
    --since)       SINCE="$2"; shift 2 ;;
    --until)       UNTIL="$2"; shift 2 ;;
    -o)            OUTPUT_FILE="$2"; FOLLOW=false; shift 2 ;;
    backend|frontend|db|nginx) SERVICE="$1"; shift ;;
    error)         SERVICE="${SERVICE:-nginx}-error"; shift ;;
    -h|--help)
      echo "GIVErS ログ表示ユーティリティ"
      echo ""
      echo "使い方: ./scripts/logs.sh [オプション] [サービス] [error]"
      echo ""
      echo "サービス:"
      echo "  backend    Go API サーバー (docker compose logs)"
      echo "  frontend   Astro SSR (docker compose logs)"
      echo "  db         PostgreSQL (docker compose logs)"
      echo "  nginx      Nginx アクセスログ (ホストファイル)"
      echo "  nginx error  Nginx エラーログ (ホストファイル)"
      echo "  (なし)     全サービス (docker compose logs)"
      echo ""
      echo "オプション:"
      echo "  --remote, -r          リモートサーバーのログを表示"
      echo "  --tail N, -n N        最新 N 行を表示 (デフォルト: 50)"
      echo "  --no-follow           リアルタイム追従なし"
      echo "  --since DATETIME      指定日時以降のログ"
      echo "  --until DATETIME      指定日時以前のログ"
      echo "  -o FILE               ログをファイルに保存 (自動 no-follow)"
      echo "  -h, --help            このヘルプを表示"
      echo ""
      echo "日時フォーマット:"
      echo "  Docker (backend/frontend/db):"
      echo "    RFC3339: 2026-03-03T10:00:00"
      echo "    相対:    1h, 30m, 24h"
      echo "  Nginx:"
      echo "    combined: 03/Mar/2026:10:00"
      exit 0
      ;;
    *)
      echo "不明な引数: $1 (--help でヘルプを表示)"
      exit 1
      ;;
  esac
done

# --- ヘルパー関数 ---

# リモートまたはローカルでコマンドを実行
run_cmd() {
  if [ "$REMOTE" = true ]; then
    ssh "$REMOTE_HOST" "$@"
  else
    eval "$@"
  fi
}

# compose コマンドプレフィックス
compose_prefix() {
  if [ "$REMOTE" = true ]; then
    echo "cd $REMOTE_DIR && docker compose -f $COMPOSE_FILE"
  else
    echo "docker compose -f $COMPOSE_FILE"
  fi
}

# nginx ログファイルパス
nginx_log_path() {
  local log_type="${1:-access}"
  if [ "$REMOTE" = true ]; then
    echo "$REMOTE_DIR/logs/nginx/${log_type}.log"
  else
    echo "logs/nginx/${log_type}.log"
  fi
}

# 出力をファイルまたは stdout に送る
output_result() {
  if [ -n "$OUTPUT_FILE" ]; then
    cat > "$OUTPUT_FILE"
    local lines
    lines=$(wc -l < "$OUTPUT_FILE")
    echo "Saved ${lines} lines to ${OUTPUT_FILE}" >&2
  else
    cat
  fi
}

# --- nginx ログの日時フィルタ (awk) ---
# combined フォーマット: 03/Mar/2026:10:00:00
nginx_since_filter() {
  local since="$1"
  if [ -z "$since" ]; then
    cat
    return
  fi
  awk -v since="$since" '
    match($0, /\[([^\]]+)\]/, m) {
      ts = m[1]
      # タイムスタンプの先頭部分で比較 (辞書順で十分)
      if (ts >= since) print
    }
  '
}

nginx_until_filter() {
  local until_dt="$1"
  if [ -z "$until_dt" ]; then
    cat
    return
  fi
  awk -v until_dt="$until_dt" '
    match($0, /\[([^\]]+)\]/, m) {
      ts = m[1]
      if (ts <= until_dt) print
    }
  '
}

# --- メイン処理 ---

case "$SERVICE" in
  nginx|nginx-error)
    # nginx ログはホストファイルから取得
    local_log_type="access"
    [ "$SERVICE" = "nginx-error" ] && local_log_type="error"
    LOG_FILE=$(nginx_log_path "$local_log_type")

    if [ -n "$SINCE" ] || [ -n "$UNTIL" ]; then
      # 日時フィルタ付き — ファイル全体を読んでフィルタ
      run_cmd "cat $LOG_FILE" \
        | nginx_since_filter "$SINCE" \
        | nginx_until_filter "$UNTIL" \
        | tail -n "$TAIL_LINES" \
        | output_result
    elif [ -n "$OUTPUT_FILE" ]; then
      # ファイル出力
      run_cmd "tail -n $TAIL_LINES $LOG_FILE" | output_result
    elif [ "$FOLLOW" = true ]; then
      run_cmd "tail -f $LOG_FILE"
    else
      run_cmd "tail -n $TAIL_LINES $LOG_FILE"
    fi
    ;;

  backend|frontend|db)
    # Docker compose logs
    PREFIX=$(compose_prefix)
    FLAGS=""
    [ "$FOLLOW" = true ] && FLAGS="-f"
    [ -n "$SINCE" ] && FLAGS="$FLAGS --since=$SINCE"
    [ -n "$UNTIL" ] && FLAGS="$FLAGS --until=$UNTIL"

    if [ -n "$OUTPUT_FILE" ]; then
      run_cmd "$PREFIX logs --tail=$TAIL_LINES $FLAGS $SERVICE" 2>&1 | output_result
    else
      run_cmd "$PREFIX logs --tail=$TAIL_LINES $FLAGS $SERVICE"
    fi
    ;;

  "")
    # 全サービス
    PREFIX=$(compose_prefix)
    FLAGS=""
    [ "$FOLLOW" = true ] && FLAGS="-f"
    [ -n "$SINCE" ] && FLAGS="$FLAGS --since=$SINCE"
    [ -n "$UNTIL" ] && FLAGS="$FLAGS --until=$UNTIL"

    if [ -n "$OUTPUT_FILE" ]; then
      run_cmd "$PREFIX logs --tail=$TAIL_LINES $FLAGS" 2>&1 | output_result
    else
      run_cmd "$PREFIX logs --tail=$TAIL_LINES $FLAGS"
    fi
    ;;
esac
