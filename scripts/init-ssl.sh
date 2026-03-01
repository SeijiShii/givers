#!/bin/bash
set -e

# GIVErS SSL初期設定スクリプト
# 使い方: ./scripts/init-ssl.sh

if [ ! -f .env ]; then
  echo "ERROR: .env が見つかりません。.env.prod.example をコピーして .env を作成してください。"
  exit 1
fi

source .env

if [ -z "$DOMAIN" ] || [ -z "$CERTBOT_EMAIL" ]; then
  echo "ERROR: .env に DOMAIN と CERTBOT_EMAIL を設定してください。"
  exit 1
fi

echo "=== Step 1: HTTP-only nginx で起動 ==="
cp nginx/conf.d/default.conf nginx/conf.d/default.conf.bak 2>/dev/null || true
cp nginx/conf.d/default.conf.initial nginx/conf.d/default.conf

docker compose -f docker-compose.prod.yml up -d nginx

echo "=== Step 2: SSL証明書を取得 ==="
docker compose -f docker-compose.prod.yml run --rm certbot certonly \
  --webroot \
  --webroot-path=/var/www/certbot \
  -d "$DOMAIN" \
  -d "www.$DOMAIN" \
  --email "$CERTBOT_EMAIL" \
  --agree-tos \
  --no-eff-email

echo "=== Step 3: HTTPS nginx設定に切り替え ==="
cp nginx/conf.d/default.conf.bak nginx/conf.d/default.conf 2>/dev/null || \
  git checkout nginx/conf.d/default.conf

echo "=== Step 4: 全サービスを起動 ==="
docker compose -f docker-compose.prod.yml up -d

echo "=== 完了 ==="
echo "https://$DOMAIN でアクセスできるか確認してください。"
echo ""
echo "証明書の自動更新を cron に追加してください:"
echo "  0 3 * * * cd $(pwd) && docker compose -f docker-compose.prod.yml run --rm certbot renew && docker compose -f docker-compose.prod.yml exec nginx nginx -s reload"
