#!/bin/bash
set -e

# GIVErS DEV環境デプロイスクリプト
# ローカルでパッケージを作成し、SCP でDEVサーバーに転送してデプロイする
#
# 使い方:
#   ./scripts/deploy-dev.sh              # 通常デプロイ (main ブランチ)
#   ./scripts/deploy-dev.sh feature/xxx  # 指定ブランチ
#   ./scripts/deploy-dev.sh --init       # 初回: ファイル転送のみ（ビルド・起動しない）

BRANCH="main"
INIT_ONLY=false

for arg in "$@"; do
  case "$arg" in
    --init) INIT_ONLY=true ;;
    *) BRANCH="$arg" ;;
  esac
done

REMOTE_HOST="givers-conoha-dev"
REMOTE_DIR="/opt/givers-dev"
LOCAL_TMP="/tmp/givers-deploy-dev"
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "=== GIVErS DEV Deploy ==="
echo "Branch: $BRANCH"
echo "Mode: $([ "$INIT_ONLY" = true ] && echo '初回（転送のみ）' || echo '通常（転送+ビルド+起動）')"
echo "Remote: $REMOTE_HOST:$REMOTE_DIR"
echo ""

# --- Step 1: ローカルでデプロイパッケージを作成 ---
echo "=== Step 1: デプロイパッケージを作成 ==="
rm -rf "$LOCAL_TMP"
mkdir -p "$LOCAL_TMP"

cd "$PROJECT_ROOT"
git checkout "$BRANCH"
git pull origin "$BRANCH"

# 必要なファイルのみコピー
rsync -a --exclude='node_modules' \
         --exclude='.git' \
         --exclude='frontend/dist' \
         --exclude='backend/tmp' \
         --exclude='.env' \
         --exclude='.env.prod' \
         --exclude='.env.dev' \
         --exclude='*.test.*' \
         --exclude='e2e/' \
         --exclude='playwright*' \
         backend/ "$LOCAL_TMP/backend/"

rsync -a --exclude='node_modules' \
         --exclude='.git' \
         --exclude='dist' \
         --exclude='.env' \
         --exclude='.env.prod' \
         --exclude='e2e/' \
         --exclude='playwright*' \
         frontend/ "$LOCAL_TMP/frontend/"

# DEV用環境変数ファイルをコピー（存在する場合）
cp frontend/.env.dev "$LOCAL_TMP/frontend/" 2>/dev/null || true

cp docker-compose.dev.yml "$LOCAL_TMP/"
cp -r nginx "$LOCAL_TMP/"
cp -r scripts "$LOCAL_TMP/"
cp -r logrotate "$LOCAL_TMP/"
cp .env.dev.example "$LOCAL_TMP/"

echo "  パッケージサイズ: $(du -sh "$LOCAL_TMP" | cut -f1)"

# --- Step 2: サーバーに転送 ---
echo ""
echo "=== Step 2: サーバーに転送 ==="
ssh "$REMOTE_HOST" "mkdir -p $REMOTE_DIR"
rsync -azP --delete \
  --exclude='.env' \
  --exclude='logs' \
  "$LOCAL_TMP/" "$REMOTE_HOST:$REMOTE_DIR/"

echo "  転送完了"

# --- Step 2.5: ログディレクトリと logrotate 設定 ---
echo ""
echo "=== Step 2.5: ログ設定 ==="
ssh "$REMOTE_HOST" "mkdir -p $REMOTE_DIR/logs/nginx"
ssh "$REMOTE_HOST" "cp $REMOTE_DIR/logrotate/nginx /etc/logrotate.d/givers-dev-nginx 2>/dev/null || true"
echo "  ログディレクトリ作成・logrotate 設定完了"

if [ "$INIT_ONLY" = true ]; then
  echo ""
  echo "=== 初回転送完了 ==="
  echo "次の手順:"
  echo "  1. ssh $REMOTE_HOST"
  echo "  2. cd $REMOTE_DIR"
  echo "  3. cp .env.dev.example .env && nano .env && chmod 600 .env"
  echo "  4. ./scripts/init-ssl-dev.sh"
  rm -rf "$LOCAL_TMP"
  exit 0
fi

# --- Step 3: サーバーでビルド・起動 ---
echo ""
echo "=== Step 3: サーバーでビルド・起動 ==="
ssh "$REMOTE_HOST" "cd $REMOTE_DIR && docker compose -f docker-compose.dev.yml up -d --build"

# --- Step 3.5: nginx 再起動（upstream の DNS キャッシュをリセット） ---
echo ""
echo "=== Step 3.5: nginx 再起動 ==="
ssh "$REMOTE_HOST" "cd $REMOTE_DIR && docker compose -f docker-compose.dev.yml restart nginx"

# --- Step 4: DBマイグレーション ---
echo ""
echo "=== Step 4: DBマイグレーション ==="
ssh "$REMOTE_HOST" "cd $REMOTE_DIR && docker compose -f docker-compose.dev.yml exec backend ./migrate"

# --- Step 5: ログヘルスチェック ---
echo ""
echo "=== Step 5: ログヘルスチェック ==="
ssh "$REMOTE_HOST" "cd $REMOTE_DIR && docker compose -f docker-compose.dev.yml logs --tail=3 backend 2>&1 | head -5"
ssh "$REMOTE_HOST" "ls -la $REMOTE_DIR/logs/nginx/ 2>/dev/null || echo 'nginx ログは初回リクエスト時に作成されます'"

echo ""
echo "=== DEV デプロイ完了 ==="
echo "https://dev.givers.work でアクセスできるか確認してください"

# クリーンアップ
rm -rf "$LOCAL_TMP"
