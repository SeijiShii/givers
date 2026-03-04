#!/bin/bash
set -e

# GIVErS DEV環境DBダンプスクリプト
# DEVサーバーの PostgreSQL をダンプし、ローカルにダウンロードする
#
# 使い方:
#   ./scripts/dump-dev-db.sh                    # カレントディレクトリに出力
#   ./scripts/dump-dev-db.sh /path/to/output    # 出力先指定

REMOTE_HOST="givers-conoha-dev"
REMOTE_DIR="/opt/givers-dev"
COMPOSE_FILE="docker-compose.dev.yml"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DUMP_FILENAME="givers_dev_${TIMESTAMP}.sql"

# 出力先
LOCAL_OUTPUT_DIR="${1:-.}"
LOCAL_DUMP_PATH="${LOCAL_OUTPUT_DIR}/${DUMP_FILENAME}"

echo "=== GIVErS DEV DBダンプ ==="
echo "リモート: ${REMOTE_HOST}:${REMOTE_DIR}"
echo "出力先:   ${LOCAL_DUMP_PATH}"
echo ""

# リモートでダンプ → ローカルにストリーム (リモートにファイルを残さない)
echo "=== ダンプ実行中 ==="
ssh "$REMOTE_HOST" "cd $REMOTE_DIR && \
  docker compose -f $COMPOSE_FILE exec -T db \
  pg_dump -U givers -d givers --no-owner --no-privileges \
  --exclude-table-data=sessions" \
  > "$LOCAL_DUMP_PATH"

echo "=== ダンプ完了 ==="
echo "ファイル: ${LOCAL_DUMP_PATH}"
echo "サイズ:   $(du -sh "$LOCAL_DUMP_PATH" | cut -f1)"
echo ""
echo "ローカルDBへの復元:"
echo "  psql -U givers -d givers < ${LOCAL_DUMP_PATH}"
