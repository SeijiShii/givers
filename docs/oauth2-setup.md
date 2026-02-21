# OAuth2 プロバイダー設定手順

本ドキュメントは、GIVErS バックエンドで各 OAuth2 プロバイダーを有効化するための設定手順をまとめます。

---

## 環境変数の全体像

`.env` ファイル（または本番環境のシークレット管理）に以下を設定します。

```env
# バックエンドの公開 URL（コールバック URL の生成に使用）
BACKEND_URL=https://api.example.com

# フロントエンドの公開 URL（認証後リダイレクト先）
FRONTEND_URL=https://example.com

# ---- Google（必須） ----
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...

# ---- GitHub（オプション） ----
GITHUB_CLIENT_ID=...
GITHUB_CLIENT_SECRET=...

# ---- Apple（オプション・将来実装） ----
APPLE_CLIENT_ID=...
APPLE_CLIENT_SECRET=...
APPLE_TEAM_ID=...
APPLE_KEY_ID=...

# ---- メールログイン（オプション・将来実装） ----
ENABLE_EMAIL_LOGIN=true
```

各プロバイダーのコールバック URL（`BACKEND_URL` に続けて設定）：

| プロバイダー | コールバック URL |
|---|---|
| Google | `{BACKEND_URL}/api/auth/google/callback` |
| GitHub | `{BACKEND_URL}/api/auth/github/callback` |
| Apple | `{BACKEND_URL}/api/auth/apple/callback` |

---

## 1. Google OAuth2（必須）

Google ログインは必須です。`GOOGLE_CLIENT_ID` が未設定の場合、サーバー起動時にエラーになります。

### 手順

1. [Google Cloud Console](https://console.cloud.google.com/) にアクセスし、プロジェクトを作成（または既存を選択）

2. **APIとサービス → 認証情報** を開き、**認証情報を作成 → OAuth クライアント ID** をクリック

3. **アプリケーションの種類** で「ウェブ アプリケーション」を選択

4. **承認済みの JavaScript 生成元** にフロントエンド URL を追加：
   ```
   https://example.com
   ```

5. **承認済みのリダイレクト URI** にコールバック URL を追加：
   ```
   https://api.example.com/api/auth/google/callback
   ```
   ローカル開発用にも追加しておく：
   ```
   http://localhost:8080/api/auth/google/callback
   ```

6. 「作成」をクリック → **クライアント ID** と **クライアントシークレット** をコピー

7. **OAuth 同意画面** を設定：
   - ユーザーの種類：外部（公開サービスの場合）
   - アプリ名・サポートメール・デベロッパー連絡先を入力
   - スコープ：`email`、`profile` を追加
   - 本番公開する場合は「アプリを公開」ボタンで審査申請（`email`/`profile` スコープは審査不要）

8. `.env` に設定：
   ```env
   GOOGLE_CLIENT_ID=xxxxxxxxxxxx.apps.googleusercontent.com
   GOOGLE_CLIENT_SECRET=GOCSPX-xxxxxxxxxxxxxxxxxx
   ```

---

## 2. GitHub OAuth2（オプション）

`GITHUB_CLIENT_ID` が設定されている場合のみ、`GET /api/auth/providers` レスポンスに `github` が含まれ、フロントエンドに GitHub ログインボタンが表示されます。

### 手順

1. GitHub にログインし、右上メニュー → **Settings** → 左サイドバー下部 **Developer settings** → **OAuth Apps** を開く

2. **New OAuth App** をクリック

3. 以下を入力：

   | 項目 | 値 |
   |---|---|
   | Application name | GIVErS（任意） |
   | Homepage URL | `https://example.com` |
   | Authorization callback URL | `https://api.example.com/api/auth/github/callback` |

4. 「Register application」をクリック → **Client ID** をコピー

5. **Generate a new client secret** をクリック → **Client Secret** をコピー（一度しか表示されません）

6. `.env` に設定：
   ```env
   GITHUB_CLIENT_ID=Iv1.xxxxxxxxxxxxxxxx
   GITHUB_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
   ```

> **スコープ:** `read:user`、`user:email`（バックエンドがリクエスト。メールアドレスが非公開設定のユーザーも取得できます）

---

## 3. Apple Sign In（オプション・将来実装）

`APPLE_CLIENT_ID` が設定されている場合のみ有効になります（現時点では将来実装のプレースホルダー）。

### 前提条件

- Apple Developer Program への加入（有料: 年額 $99）
- macOS / iOS アプリとは独立して Web のみの設定も可能

### 手順

1. [Apple Developer](https://developer.apple.com/account/) にログイン

2. **Certificates, Identifiers & Profiles → Identifiers** を開き、**App IDs** を作成：
   - Description: GIVErS
   - Bundle ID: `com.example.givers`（任意の逆ドメイン形式）
   - Capabilities: **Sign In with Apple** にチェック

3. 同じく **Identifiers → Services IDs** を作成：
   - Description: GIVErS Web
   - Identifier: `com.example.givers.web`（これが `APPLE_CLIENT_ID` になる）
   - **Sign In with Apple** を有効化し「Configure」をクリック：
     - Primary App ID: 先ほど作成した App ID を選択
     - Domains: `api.example.com`
     - Return URLs: `https://api.example.com/api/auth/apple/callback`

4. **Keys** で新しいキーを作成：
   - Key Name: GIVErS Sign In with Apple
   - **Sign In with Apple** にチェック → Configure → Primary App ID を選択
   - 作成後、`.p8` ファイル（秘密鍵）をダウンロード（一度のみ）
   - **Key ID** をメモ

5. Team ID は Apple Developer の右上メニューまたはメンバーシップ画面で確認

6. `.p8` ファイルの内容を Client Secret として設定（または別途 JWT 生成）：
   ```env
   APPLE_CLIENT_ID=com.example.givers.web
   APPLE_TEAM_ID=XXXXXXXXXX
   APPLE_KEY_ID=XXXXXXXXXX
   APPLE_CLIENT_SECRET=<.p8 ファイルの内容、または生成した JWT>
   ```

> **注意:** Apple の Client Secret は `.p8` 秘密鍵から JWT を生成して使用するのが一般的です。詳細は [Apple のドキュメント](https://developer.apple.com/documentation/sign_in_with_apple/generate_and_validate_tokens) を参照してください。

---

## 4. メールログイン（オプション・将来実装）

マジックリンク方式（メールアドレス入力 → リンクをメール送信 → クリックでログイン）の実装です。

```env
ENABLE_EMAIL_LOGIN=true
```

メール送信には SMTP または SendGrid 等の設定が別途必要です（実装時に追記予定）。

---

## ローカル開発の設定例

`backend/.env`（`.gitignore` 済み）：

```env
DATABASE_URL=postgres://givers:givers@localhost:5432/givers?sslmode=disable
BACKEND_URL=http://localhost:8080
FRONTEND_URL=http://localhost:4321
SESSION_SECRET=dev-secret-change-in-production-32bytes

GOOGLE_CLIENT_ID=xxxxxxxxxxxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-xxxxxxxxxxxxxxxxxx

# GitHub はオプション。設定しなければ GitHub ボタン非表示
# GITHUB_CLIENT_ID=
# GITHUB_CLIENT_SECRET=
```

---

## プロバイダーの有効・無効の確認

バックエンド起動後、以下で現在有効なプロバイダーを確認できます：

```bash
curl http://localhost:8080/api/auth/providers
# => {"providers":["google","github"]}
```

フロントエンドはこのレスポンスに基づいてログインボタンを動的に表示します。
