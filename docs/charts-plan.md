# チャート表示プラン

プロジェクトページとマイページで推移・進捗をチャート表示する機能の実装プラン。

## 概要

| ページ | チャート種別 | 表示内容 |
|--------|--------------|----------|
| プロジェクトページ | 折れ線グラフ | 目標金額、実際の寄付額、費やしたコストの時系列推移 |
| マイページ | 複数チャート | 月ごと合計寄付額、プロジェクト別寄付額 |

## 前提

- Phase 3（プロジェクト CRUD）、Phase 4（決済）完了後に実装
- 寄付・コストの時系列データが DB に蓄積されていること

---

## プロジェクトページ

### 折れ線グラフで表示するデータ

| 系列 | 説明 | データソース |
|------|------|--------------|
| 目標金額 | 月額目標の累積または月別 | project_costs（月額目標の算出） |
| 実際の寄付額 | 月別の寄付合計 | donations（created_at で集計） |
| 費やしたコスト | 月別の実績コスト | project_cost_records（実績記録テーブル） |

### 時系列の粒度

- 月単位（デフォルト）
- 表示期間: 直近 6 ヶ月 / 1 年 / 全期間（切り替え可能）

### 想定 API

```
GET /api/projects/:id/chart
  ?period=6m|1y|all
  ?granularity=month

Response:
{
  "labels": ["2025-01", "2025-02", ...],
  "series": [
    { "name": "目標金額", "data": [50000, 50000, ...] },
    { "name": "寄付額", "data": [42000, 45000, ...] },
    { "name": "費やしたコスト", "data": [38000, 40000, ...] }
  ]
}
```

---

## マイページ

### 表示するチャート

1. **月ごと合計寄付額**
   - 棒グラフまたは折れ線
   - 横軸: 月、縦軸: 寄付額（円）
   - 自分が寄付した総額の月別推移

2. **プロジェクト別寄付額**
   - 円グラフまたは棒グラフ
   - 支援したプロジェクトごとの寄付額の内訳
   - 例: プロジェクトA 60%、プロジェクトB 30%、プロジェクトC 10%

3. **サマリー**
   - 合計寄付額
   - 支援プロジェクト数
   - 直近の寄付一覧

### 想定 API

```
GET /api/me/donations/chart
  ?period=6m|1y|all

Response:
{
  "monthly": [
    { "month": "2025-01", "amount": 15000 },
    { "month": "2025-02", "amount": 22000 },
    ...
  ],
  "byProject": [
    { "projectId": "...", "projectName": "...", "amount": 50000 },
    ...
  ],
  "total": 120000,
  "projectCount": 3
}
```

---

## 技術選定

### チャートライブラリ（React）

| 候補 | 特徴 | サイズ |
|------|------|--------|
| **Recharts** | React 向け、宣言的、軽量 | ~100KB |
| Chart.js + react-chartjs-2 | 汎用、柔軟 | ~60KB |
| Victory | アニメーション豊富 | やや大 |

**推奨: Recharts**
- React との相性が良い
- 折れ線・棒・円グラフを統一 API で扱える
- 型定義あり（TypeScript）

### 導入

```bash
npm install recharts
```

---

## コンポーネント構成

```
frontend/src/components/react/
├── charts/
│   ├── ProjectChart.tsx      # プロジェクトページ用折れ線グラフ
│   ├── MonthlyDonationChart.tsx  # マイページ: 月別寄付
│   └── DonationByProjectChart.tsx # マイページ: プロジェクト別
```

---

## データモデル（追加・拡張）

### project_cost_records（実績コスト記録）

| カラム | 型 | 説明 |
|--------|-----|------|
| id | UUID | |
| project_id | UUID | |
| month | DATE | 対象月 |
| server_cost | INT | 実績サーバー費用 |
| dev_cost | INT | 実績開発費 |
| other_cost | INT | その他 |
| created_at | TIMESTAMP | |

※ 計画値は project_costs、実績値は project_cost_records で管理

### donations の拡張

- created_at で月別集計可能であること
- project_id でプロジェクト別集計可能であること

---

## 実装順序

1. **API 実装**（Phase 4 完了後）
   - GET /api/projects/:id/chart
   - GET /api/me/donations/chart

2. **Recharts 導入**
   - package.json に追加
   - 共通のチャートスタイル（草原テーマに合わせた色）

3. **プロジェクトページ**
   - ProjectChart コンポーネント
   - 期間・粒度の切り替え UI

4. **マイページ**
   - MonthlyDonationChart
   - DonationByProjectChart
   - サマリー表示

5. **手動動作確認**
   - データ投入後のチャート表示
   - 空データ時の表示
   - レスポンシブ対応

---

## 手動動作確認チェックリスト（チャート）

### プロジェクトページ

- [ ] 折れ線グラフに目標・寄付・コストの 3 系列が表示される
- [ ] 期間切り替え（6m/1y/all）でデータが更新される
- [ ] データがない場合は適切なメッセージが表示される
- [ ] 草原テーマの色（緑・黄）で統一されている

### マイページ

- [ ] 月別寄付額チャートが表示される
- [ ] プロジェクト別寄付額の内訳が表示される
- [ ] 合計・支援プロジェクト数が表示される
- [ ] 寄付履歴がない場合は適切なメッセージが表示される
