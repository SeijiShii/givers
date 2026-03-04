#!/bin/bash
set -e

# GIVErS DEV環境 SSL初期設定スクリプト
# 使い方: ./scripts/init-ssl-dev.sh

if [ ! -f .env ]; then
  echo "ERROR: .env が見つかりません。.env.dev.example をコピーして .env を作成してください。"
  exit 1
fi

source .env

if [ -z "$DOMAIN" ] || [ -z "$CERTBOT_EMAIL" ]; then
  echo "ERROR: .env に DOMAIN と CERTBOT_EMAIL を設定してください。"
  exit 1
fi

echo "=== DEV SSL Setup: $DOMAIN ==="

echo "=== Step 1: HTTP-only nginx で起動 ==="
cp nginx/conf.d/default.conf.dev nginx/conf.d/default.conf.dev.bak 2>/dev/null || true
cp nginx/conf.d/default.conf.dev.initial nginx/conf.d/default.conf.dev

docker compose -f docker-compose.dev.yml up -d nginx

echo "=== Step 2: SSL証明書を取得 ==="
docker compose -f docker-compose.dev.yml run --rm certbot certonly \
  --webroot \
  --webroot-path=/var/www/certbot \
  -d "$DOMAIN" \
  --email "$CERTBOT_EMAIL" \
  --agree-tos \
  --no-eff-email

echo "=== Step 3: HTTPS nginx設定に切り替え ==="
cp nginx/conf.d/default.conf.dev.bak nginx/conf.d/default.conf.dev 2>/dev/null || \
  echo "WARNING: バックアップが見つかりません。default.conf.dev を手動で復元してください。"

echo "=== Step 4: 全サービスを起動 ==="
docker compose -f docker-compose.dev.yml up -d --build

echo "=== Step 5: nginx に SSL 設定を反映 ==="
docker compose -f docker-compose.dev.yml exec nginx nginx -s reload

echo "=== Step 6: DBマイグレーション ==="
docker compose -f docker-compose.dev.yml exec backend ./migrate

echo "=== 完了 ==="
echo "https://$DOMAIN でアクセスできるか確認してください。"
echo ""
echo "証明書の自動更新を cron に追加してください:"
echo "  (echo \"0 3 * * * cd /opt/givers-dev && docker compose -f docker-compose.dev.yml run --rm certbot renew --quiet && docker compose -f docker-compose.dev.yml exec nginx nginx -s reload\") | crontab -"
