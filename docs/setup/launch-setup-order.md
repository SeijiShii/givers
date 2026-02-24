# 本番リリース前の外部サービス設定順序

## なぜ順番が重要か

各ステップに依存関係があるため、順番を誤ると後から設定をやり直す必要が生じる。

```
ドメイン取得
    │
    ├──→ Stripe 設定（Webhook URL にドメインを使う）
    │         │
    │         └──→ Webhook URL 登録（https://your-domain/api/webhooks/stripe）
    │
    └──→ ConoHa VPS 契約
              │
              ├──→ DNS: A レコードをVPS IPに向ける（ドメインが必要）
              └──→ SSL 証明書取得（certbot、ドメインが必要）
```

**推奨順序：ドメイン → Stripe → ConoHa**

---

## Step 1：ドメイン取得

**理由：** Stripe の Connect 設定と ConoHa の DNS 設定の両方にドメイン名が必要。
サーバー契約前でも取得できるため、最初に済ませる。

### 作業内容

- [ ] ドメインを取得する（お名前.com / Cloudflare / ConoHa ドメイン等）
  - 例: `givers.work`
  - ConoHa でドメインも取る場合は Step 3 と同時でも可
- [ ] 取得したドメイン名をメモしておく（以降の設定で繰り返し使う）

### ポイント

- **Cloudflare 経由で購入すると DNS 管理が楽**。後述の A レコード設定や SSL も Cloudflare 任せにできる。
- `.com` `.jp` などは価格差が大きい。ランニングコストを確認してから購入する。

---

## Step 2：Stripe アカウント設定

**理由：** Stripe Webhook URL の登録にドメインが必要。
ConoHa でアプリを動かす前に Stripe 側の設定を完成させておくと、
デプロイ後すぐに決済テストができる。

### 2-1. Stripe アカウント登録・本人確認

- [ ] [stripe.com](https://stripe.com/jp) でアカウントを作成
- [ ] ビジネス情報・本人確認を完了させる
  - 本番 API キーを使うには本人確認が必須
  - 個人の場合：個人事業主として登録
- [ ] テストモードと本番モードを切り替えるキーをそれぞれ取得

### 2-2. Connect の設定（Accounts v2 API）

プロジェクトオーナー用の連結アカウントを作成する機能の設定。
Accounts v2 API を使用しており、OAuth や `STRIPE_CONNECT_CLIENT_ID` は不要。

- [ ] Stripe ダッシュボード → **Connect** → 設定を開く
- [ ] **プラットフォーム名・アイコン** を設定する（オーナーのオンボーディング画面に表示される）

> **Note:** v2 API ではプラットフォームがアカウントを作成し、Account Links でオンボーディングを行う。
> `STRIPE_SECRET_KEY` のみで連結アカウントの作成が可能。

### 2-3. Webhook エンドポイントの登録

- [ ] Stripe ダッシュボード → **開発者** → **Webhook** → エンドポイントを追加
  ```
  https://your-domain.example.com/api/webhooks/stripe
  ```
- [ ] 受信するイベントを選択（最低限）
  - `payment_intent.succeeded`
  - `payment_intent.payment_failed`
  - `customer.subscription.created`
  - `customer.subscription.deleted`
  - `account.updated`（Connect 利用時）
- [ ] 署名シークレット（`whsec_...`）をメモ → `STRIPE_WEBHOOK_SECRET`

> **注意：** この時点ではサーバーがまだ動いていないため、Webhook の疎通確認は Step 3 完了後に行う。

### 2-4. 環境変数の整理

この時点で `.env`（本番用）に以下を記録できる状態になる：

```env
STRIPE_SECRET_KEY=sk_live_...
STRIPE_PUBLISHABLE_KEY=pk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...
```

> テストモードのキー（`sk_test_...`）と本番キー（`sk_live_...`）を混在させない。
> 開発環境は `sk_test_`、本番サーバーは `sk_live_` を使う。

---

## Step 3：ConoHa VPS 契約・初期設定

**理由：** Stripe の設定が揃ってから環境変数をまとめてサーバーに投入できる。
ドメインの A レコードは VPS の IP を知らないと設定できないため、ここで行う。

### 3-1. VPS 契約

- [ ] [ConoHa VPS](https://www.conoha.jp/vps/) で契約
  - OS: Ubuntu 22.04 LTS を選択
  - メモリ: 1GB〜（DB・アプリ同居の場合）
- [ ] root パスワードまたは SSH 鍵を設定
- [ ] **VPS のグローバル IP アドレス** をメモ

### 3-2. DNS 設定（ドメインを VPS に向ける）

- [ ] ドメインの管理画面（お名前.com / Cloudflare 等）を開く
- [ ] A レコードを追加
  ```
  your-domain.example.com  →  （VPS の IP アドレス）
  ```
- [ ] 反映を待つ（数分〜24 時間。Cloudflare なら数秒）
- [ ] 反映確認：`ping your-domain.example.com` または `nslookup`

### 3-3. サーバー初期設定

詳細は [conoha-deployment.md](conoha-deployment.md) を参照。概要のみ記載。

- [ ] SSH 接続
- [ ] sudo ユーザー作成・root ログイン無効化
- [ ] SSH 鍵認証に変更・パスワード認証無効化
- [ ] UFW でポート開放（22 / 80 / 443）
- [ ] Docker / Docker Compose インストール

### 3-4. アプリのデプロイ

- [ ] リポジトリを clone
- [ ] `.env`（本番用）を配置（`chmod 600`）
  - Step 2-4 で整理した Stripe キー群を含む
  - その他の環境変数（`DATABASE_URL`, `SESSION_SECRET`, OAuth キー等）も設定
- [ ] `docker compose -f docker-compose.prod.yml up -d` で起動

### 3-5. SSL 証明書の取得

DNS が向いていることを確認してから実行する。

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d your-domain.example.com
```

- [ ] 証明書取得完了
- [ ] HTTPS でサービスにアクセスできることを確認

---

## Step 4：疎通確認チェックリスト

全設定完了後に確認する項目。

- [ ] `https://your-domain.example.com` でフロントが表示される
- [ ] `https://your-domain.example.com/api/health` が 200 を返す
- [ ] Stripe テストモードで寄付フローが通る
- [ ] Stripe ダッシュボードで Webhook の受信ログが確認できる（`200 OK`）
- [ ] Google / GitHub OAuth でのログインが通る（コールバック URL が正しく設定されているか）
- [ ] プロジェクトオーナーの Stripe オンボーディングフローが通る

---

## 補足：テスト → 本番の切り替え

| 段階 | Stripe キー | 目的 |
|---|---|---|
| 開発・ステージング | `sk_test_...` / `pk_test_...` | 決済を実際に課金しない |
| 本番リリース | `sk_live_...` / `pk_live_...` | 実際の課金が発生する |

本番キーへの切り替えは、**テストで全フローが通ってから** 行う。
Webhook エンドポイントもテスト用と本番用で別に登録しておくと管理しやすい。

---

## 関連ドキュメント

- [stripe-connect-setup.md](stripe-connect-setup.md) - Stripe Connect ホストアカウント設定ガイド（詳細）
- [conoha-deployment.md](conoha-deployment.md) - サーバー設定の詳細手順
- [oauth2-setup.md](oauth2-setup.md) - Google / GitHub OAuth の設定
- [implementation-plan.md](../design/implementation-plan.md) - 環境変数一覧
