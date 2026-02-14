# GIVErS

**GIVEの精神** / Donation platform for the GIVE spirit.

## コンセプト

- **作り手の GIVE**: 見返りを求めず、自分が良いと思うものを作る
- **受け手の GIVE**: 使って良かった人が、自発的に応援したいと思う
- **ゼロ手数料**: GIVErS は手数料を取らない
- **透明性**: プロジェクトの継続性や費用を定量化して公開

## 概要

クリエイターが無料で提供し、利用者が「応援したい」と思ったときに寄付できるプラットフォーム。  
クラウドファンディングのような目標設定やリワードは不要。  
プロジェクトの月額コストや達成率を透明に表示し、寄付者が判断しやすくする。

## 開発

### 起動

```bash
docker compose up -d db
cd backend && go run ./cmd/migrate   # 初回のみ
cd backend && go run ./cmd/server
cd frontend && npm run dev
```

### OAuth（Phase 2）

- **Google**: [Google Cloud Console](https://console.cloud.google.com/) で OAuth 2.0 クライアント ID を作成。リダイレクト URI: `http://localhost:8080/api/auth/google/callback`
- **GitHub**: [GitHub Developer Settings](https://github.com/settings/developers) で OAuth App を作成。Authorization callback URL: `http://localhost:8080/api/auth/github/callback`

## 詳細

詳しい思想や仕様は [docs/idea.md](docs/idea.md) を参照してください。

## License

Apache-2.0
