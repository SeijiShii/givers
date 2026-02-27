# OAuth2 プロバイダー設定手順

GIVErS バックエンドで各 OAuth2 プロバイダーを有効化するための詳細手順です。

---

## 目次

- [環境変数一覧](#環境変数一覧)
- [コールバック URL 一覧](#コールバック-url-一覧)
- [1. Google OAuth2（必須）](#1-google-oauth2必須)
- [2. GitHub OAuth2（オプション）](#2-github-oauth2オプション)
- [3. Discord OAuth2（オプション）](#3-discord-oauth2オプション)
- [4. Apple Sign In（オプション・将来実装）](#4-apple-sign-inオプション将来実装)
- [5. メールログイン（オプション・将来実装）](#5-メールログインオプション将来実装)
- [ローカル開発の設定例](#ローカル開発の設定例)
- [動作確認](#動作確認)
- [トラブルシューティング](#トラブルシューティング)

---

## 環境変数一覧

`backend/.env`（`.gitignore` 済み）に設定します。

```env
# ---- サーバー ----
DATABASE_URL=postgres://givers:givers@localhost:5432/givers?sslmode=disable
BACKEND_URL=https://api.example.com     # コールバック URL の生成に使用
FRONTEND_URL=https://example.com        # 認証後のリダイレクト先
SESSION_SECRET=<32文字以上のランダム文字列>

# ---- Google（必須） ----
GOOGLE_CLIENT_ID=xxxxxxxxxxxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-xxxxxxxxxxxxxxxxxx

# ---- GitHub（オプション: 設定しなければ GitHub ボタン非表示） ----
GITHUB_CLIENT_ID=Iv1.xxxxxxxxxxxxxxxx
GITHUB_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# ---- Discord（オプション: 設定しなければ Discord ボタン非表示） ----
DISCORD_CLIENT_ID=xxxxxxxxxxxxxxxxxxxx
DISCORD_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# ---- Apple（オプション・将来実装） ----
APPLE_CLIENT_ID=com.example.givers.web
APPLE_TEAM_ID=XXXXXXXXXX
APPLE_KEY_ID=XXXXXXXXXX
APPLE_CLIENT_SECRET=-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----

# ---- メールログイン（オプション・将来実装） ----
ENABLE_EMAIL_LOGIN=true
```

## コールバック URL 一覧

各プロバイダーのコンソールに登録するコールバック URL です。

| プロバイダー | 本番 | ローカル開発 |
|---|---|---|
| Google | `https://api.example.com/api/auth/google/callback` | `http://localhost:8080/api/auth/google/callback` |
| GitHub | `https://api.example.com/api/auth/github/callback` | `http://localhost:8080/api/auth/github/callback` |
| Discord | `https://api.example.com/api/auth/discord/callback` | `http://localhost:8080/api/auth/discord/callback` |
| Apple | `https://api.example.com/api/auth/apple/callback` | ※ Apple は HTTPS 必須。ローカルテスト不可 |

---

## 1. Google OAuth2（必須）

`GOOGLE_CLIENT_ID` が未設定の場合はサーバー起動時にエラーになります。

### 1-1. Google Cloud プロジェクトを作成する

1. [Google Cloud Console](https://console.cloud.google.com/) を開く
2. 画面上部のプロジェクト選択ドロップダウン → **「新しいプロジェクト」**
3. プロジェクト名（例: `givers`）を入力 → **「作成」**
4. 作成後、そのプロジェクトを選択した状態にする

> 既存プロジェクトを使う場合はこの手順をスキップ。

### 1-2. OAuth 同意画面を設定する

認証情報を作成する前に、同意画面の設定が必要です。

1. 左メニュー **「APIとサービス」→「OAuth 同意画面」** を開く
2. ユーザーの種類：
   - **「外部」** を選択（一般公開サービスの場合）
   - 「内部」は Google Workspace 組織内のみ
3. **「作成」** をクリック
4. 以下を入力：

   | 項目 | 値 |
   |---|---|
   | アプリ名 | GIVErS |
   | ユーザーサポートメール | 自分のメールアドレス |
   | アプリのロゴ | （任意）|
   | アプリのドメイン → アプリケーションのホームページ | `https://example.com` |
   | デベロッパーの連絡先情報 | 自分のメールアドレス |

5. **「保存して次へ」**
6. スコープの追加 → **「スコープを追加または削除」**：
   - `https://www.googleapis.com/auth/userinfo.email`（`email`）
   - `https://www.googleapis.com/auth/userinfo.profile`（`profile`）
   - OpenID Connect の `openid`
   - 上記3つを選択して **「更新」**
7. **「保存して次へ」**
8. テストユーザーの追加（開発中は審査前でも追加したユーザーのみログイン可）：
   - **「+ ADD USERS」** → 自分のメールアドレスを追加
9. **「保存して次へ」** → 「ダッシュボードに戻る」

> **本番公開時:** 同意画面の **「アプリを公開」** ボタンで公開状態にします。`email` / `profile` / `openid` スコープのみであれば審査（verification）は不要です。

### 1-3. OAuth クライアント ID を作成する

1. 左メニュー **「APIとサービス」→「認証情報」** を開く
2. 上部の **「+ 認証情報を作成」→「OAuth クライアント ID」** をクリック
3. アプリケーションの種類：**「ウェブ アプリケーション」**
4. 名前：`GIVErS Web`（任意）
5. **「承認済みの JavaScript 生成元」** に追加：
   ```
   https://example.com
   http://localhost:4321
   ```
6. **「承認済みのリダイレクト URI」** に追加：
   ```
   https://api.example.com/api/auth/google/callback
   http://localhost:8080/api/auth/google/callback
   ```
7. **「作成」** をクリック
8. ダイアログに表示される **「クライアント ID」** と **「クライアントシークレット」** をコピーして保存

   ```
   クライアント ID:      xxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.apps.googleusercontent.com
   クライアントシークレット: GOCSPX-xxxxxxxxxxxxxxxxxxxxxxxxxx
   ```

   > シークレットは後から「認証情報」画面 → 鉛筆アイコンで再確認できますが、紛失した場合は再生成が必要です。

### 1-4. .env に設定する

```env
GOOGLE_CLIENT_ID=xxxxxxxxxxxx-xxxxxxxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-xxxxxxxxxxxxxxxxxxxxxxxxxx
```

---

## 2. GitHub OAuth2（オプション）

設定しない場合、`GET /api/auth/providers` レスポンスに `github` が含まれず、フロントエンドの GitHub ボタンも表示されません。

### 2-1. OAuth App を作成する

1. GitHub にログイン → 右上のアイコン → **「Settings」**
2. 左サイドバー最下部 **「Developer settings」** → **「OAuth Apps」**
3. **「New OAuth App」** をクリック
4. 以下を入力：

   | 項目 | 値 |
   |---|---|
   | Application name | GIVErS |
   | Homepage URL | `https://example.com` |
   | Application description | （任意）|
   | Authorization callback URL | `https://api.example.com/api/auth/github/callback` |

   > **ローカル開発用に別 App を作る場合:**
   > Authorization callback URL を `http://localhost:8080/api/auth/github/callback` にした App を別途作成して、ローカル用 `.env` に使います。
   > （1つの OAuth App に複数コールバックを登録する機能は GitHub にはありません）

5. **「Register application」** をクリック

### 2-2. クライアントシークレットを生成する

1. 作成後の App 詳細画面で **「Generate a new client secret」** をクリック
2. 表示されるシークレットを**すぐにコピー**（画面を離れると再表示されません）

   ```
   Client ID:     Iv1.xxxxxxxxxxxxxxxx
   Client secret: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
   ```

### 2-3. .env に設定する

```env
GITHUB_CLIENT_ID=Iv1.xxxxxxxxxxxxxxxx
GITHUB_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

> **スコープ:** バックエンドは `read:user` と `user:email` をリクエストします。`user:email` により、メールアドレスを非公開設定にしているユーザーのアドレスも取得できます。

> **Organization OAuth App:** 組織メンバー限定にしたい場合は個人設定ではなく Organization の Settings から作成します（通常の GIVErS では不要）。

---

## 3. Discord OAuth2（オプション）

設定しない場合、`GET /api/auth/providers` レスポンスに `discord` が含まれず、フロントエンドの Discord ボタンも表示されません。

### 3-1. Discord Application を作成する

1. [Discord Developer Portal](https://discord.com/developers/applications) を開く
2. **「New Application」** をクリック
3. アプリ名（例: `GIVErS`）を入力 → **「Create」**

### 3-2. OAuth2 を設定する

1. 左メニュー **「OAuth2」→「General」** を開く
2. **Client ID** をコピー（`DISCORD_CLIENT_ID` に使用）
3. **「Reset Secret」** をクリックし、表示される **Client Secret** をコピー（`DISCORD_CLIENT_SECRET` に使用。**表示は一度きり**）
4. **「Redirects」** に以下を追加：
   - ローカル: `http://localhost:8080/api/auth/discord/callback`
   - 本番: `https://api.example.com/api/auth/discord/callback`
5. **「Save Changes」** をクリック

### 3-3. .env に設定する

```env
DISCORD_CLIENT_ID=xxxxxxxxxxxxxxxxxxxx
DISCORD_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

> **スコープ:** バックエンドは `identify` と `email` をリクエストします。Discord はメールアドレスを非公開にできるため、取得できない場合は `username@discord.invalid` をフォールバックとして使用します。

> **メールによるアカウントリンク:** Discord のメールアドレスと一致する既存ユーザーがいる場合、そのユーザーに `discord_id` がリンクされます（GitHub と同じ方式）。

---

## 4. Apple Sign In（オプション・将来実装）

現時点ではバックエンドの実装が未完了のため、設定しても有効になりません。将来実装時のための手順メモです。

### 前提条件

- Apple Developer Program への加入（年額 $99 USD）
- Web のみの Sign In with Apple は iOS アプリなしで設定可能
- **コールバック URL は HTTPS 必須**（ローカル開発での直接テストは不可）

### 4-1. App ID を作成する

1. [Apple Developer](https://developer.apple.com/account/) → **「Certificates, Identifiers & Profiles」**
2. 左メニュー **「Identifiers」** → 右上 **「+」**
3. 種類：**「App IDs」** → Continue
4. タイプ：**「App」** → Continue
5. 以下を入力：
   - Description: `GIVErS`
   - Bundle ID（Explicit）: `com.example.givers`（自分のドメインに合わせて変更）
6. Capabilities リストから **「Sign In with Apple」** にチェック
7. **「Continue」→「Register」**

### 4-2. Services ID を作成する（Web 用 Client ID）

1. **「Identifiers」** → **「+」**
2. 種類：**「Services IDs」** → Continue
3. 以下を入力：
   - Description: `GIVErS Web`
   - Identifier: `com.example.givers.web`（**これが `APPLE_CLIENT_ID` になる**）
4. **「Continue」→「Register」**
5. 作成した Services ID をクリックして編集画面を開く
6. **「Sign In with Apple」** を有効化 → **「Configure」**：
   - Primary App ID: 3-1 で作成した App ID を選択
   - **Domains and Subdomains**: `api.example.com`
   - **Return URLs**: `https://api.example.com/api/auth/apple/callback`
7. **「Next」→「Done」→「Continue」→「Save」**

### 4-3. Key（秘密鍵）を作成する

1. 左メニュー **「Keys」** → **「+」**
2. Key Name: `GIVErS Sign In with Apple`
3. **「Sign In with Apple」** にチェック → **「Configure」**：
   - Primary App ID: 3-1 で作成した App ID を選択
4. **「Save」→「Continue」→「Register」**
5. **「Download」** をクリックして `.p8` ファイルをダウンロード（**一度しかダウンロードできません**）
6. **Key ID**（10文字の英数字）をメモ

### 4-4. Team ID を確認する

[Apple Developer メンバーシップ](https://developer.apple.com/account/#/membership/) を開くと **Team ID**（10文字の英数字）が表示されます。

### 4-5. Client Secret（JWT）を生成する

Apple の OAuth2 では Client Secret は固定の文字列ではなく、`.p8` 秘密鍵で署名した **JWT** を都度生成して使用します。有効期限は最大 **6ヶ月**。

```python
# generate_apple_secret.py
import jwt
import time
from pathlib import Path

TEAM_ID    = "XXXXXXXXXX"          # 3-4 で確認した Team ID
CLIENT_ID  = "com.example.givers.web"  # 3-2 で設定した Services ID
KEY_ID     = "XXXXXXXXXX"          # 3-3 で確認した Key ID
KEY_FILE   = "AuthKey_XXXXXXXXXX.p8"   # ダウンロードした .p8 ファイル

private_key = Path(KEY_FILE).read_text()

payload = {
    "iss": TEAM_ID,
    "iat": int(time.time()),
    "exp": int(time.time()) + 86400 * 180,  # 180日（最大6ヶ月）
    "aud": "https://appleid.apple.com",
    "sub": CLIENT_ID,
}

token = jwt.encode(payload, private_key, algorithm="ES256", headers={"kid": KEY_ID})
print(token)
```

```bash
pip install PyJWT cryptography
python generate_apple_secret.py
```

### 4-6. .env に設定する

```env
APPLE_CLIENT_ID=com.example.givers.web
APPLE_TEAM_ID=XXXXXXXXXX
APPLE_KEY_ID=XXXXXXXXXX
# 生成した JWT を設定（6ヶ月ごとに再生成が必要）
APPLE_CLIENT_SECRET=eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6Ii4uLiJ9...
```

---

## 5. メールログイン（オプション・将来実装）

マジックリンク方式（メールアドレス入力 → リンクをメール送信 → クリックでログイン）です。現時点ではバックエンド未実装です。

```env
ENABLE_EMAIL_LOGIN=true
```

メール送信の実装時には SMTP または SendGrid / Resend 等の設定が別途必要です。

---

## ローカル開発の設定例

`backend/.env`（`.gitignore` 済み）の最小構成：

```env
DATABASE_URL=postgres://givers:givers@localhost:5432/givers?sslmode=disable
BACKEND_URL=http://localhost:8080
FRONTEND_URL=http://localhost:4321
SESSION_SECRET=dev-secret-change-in-production-at-least-32-bytes

# Google（必須）
GOOGLE_CLIENT_ID=xxxxxxxxxxxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-xxxxxxxxxxxxxxxxxxxxxxxxxx

# GitHub（オプション）
# GITHUB_CLIENT_ID=Iv1.xxxxxxxxxxxxxxxx
# GITHUB_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Discord（オプション）
# DISCORD_CLIENT_ID=xxxxxxxxxxxxxxxxxxxx
# DISCORD_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

---

## 動作確認

バックエンド起動後、以下で有効なプロバイダーを確認できます：

```bash
curl http://localhost:8080/api/auth/providers
# Google のみの場合: {"providers":["google"]}
# GitHub も有効の場合: {"providers":["google","github"]}
# Discord も有効の場合: {"providers":["google","github","discord"]}
```

フロントエンドはこのレスポンスに基づいてログインボタンを動的に表示します。

---

## トラブルシューティング

### Google: 「このアプリはブロックされています」

同意画面のステータスが「テスト中」のとき、テストユーザーとして登録されていないアカウントでログインしようとするとこのエラーが出ます。

**対処:** 
- 開発中: 同意画面 → テストユーザーに使用するアカウントを追加
- 本番: 同意画面 → **「アプリを公開」** をクリック（`email`/`profile` スコープのみなら審査不要で即時公開）

### Google: `redirect_uri_mismatch`

コールバック URL が Cloud Console に登録されているものと一致していません。

**対処:** Cloud Console の「認証情報」→ OAuth クライアント ID を編集し、`BACKEND_URL` と一致する URL を **「承認済みのリダイレクト URI」** に追加する。

### GitHub: ログイン後に `bad_verification_code`

Client Secret が誤っているか、コールバック URL が App 設定と一致していない可能性があります。

**対処:**
1. GitHub OAuth App の設定画面で Authorization callback URL を確認
2. `.env` の `GITHUB_CLIENT_SECRET` を再確認（先頭・末尾の空白に注意）

### セッションが維持されない

`SESSION_SECRET` が起動のたびに変わっている（例: 毎回ランダム生成）と、再起動後にログアウトされます。

**対処:** `.env` に固定の `SESSION_SECRET` を設定する（32文字以上の任意の文字列）。

```bash
# ランダム生成例（一度実行してコピーして .env に貼る）
openssl rand -base64 32
```
