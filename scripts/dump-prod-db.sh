#!/bin/bash
set -e

# GIVErS 本番DBダンプスクリプト
# 本番サーバーの PostgreSQL をダンプし、ローカルにダウンロードする
#
# 使い方:
#   ./scripts/dump-prod-db.sh                    # カレントディレクトリに出力
#   ./scripts/dump-prod-db.sh /path/to/output    # 出力先指定

REMOTE_HOST="givers-conoha-root"
REMOTE_DIR="/opt/givers"
COMPOSE_FILE="docker-compose.prod.yml"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DUMP_FILENAME="givers_prod_${TIMESTAMP}.sql"

# 出力先
LOCAL_OUTPUT_DIR="${1:-.}"
LOCAL_DUMP_PATH="${LOCAL_OUTPUT_DIR}/${DUMP_FILENAME}"

echo "=== GIVErS 本番DBダンプ ==="
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
echo ""
echo "DEVサーバーへの投入:"
echo "  cat ${LOCAL_DUMP_PATH} | ssh givers-conoha-dev 'cd /opt/givers-dev && docker compose -f docker-compose.dev.yml exec -T db psql -U givers -d givers'"
