import type { Project } from "../lib/api";

const now = new Date();
const day = (d: number) => {
  const d2 = new Date(now);
  d2.setDate(d2.getDate() - d);
  return d2.toISOString();
};

/** 達成率を計算するための currentMonthly をモックで持つ */
export interface MockProject extends Project {
  _mockCurrentMonthly?: number;
  _mockImageUrl?: string;
  _mockOverview?: string;
}

export const MOCK_PROJECTS: MockProject[] = [
  {
    id: "mock-1",
    owner_id: "user-1",
    name: "オープンソースの軽量エディタ",
    description:
      "誰でも使える、シンプルで軽量なテキストエディタ。プラグインで拡張可能。",
    _mockImageUrl:
      "https://placehold.co/800x400/4a7c59/ffffff?text=Lightweight+Editor",
    _mockOverview: `## このプロジェクトについて

誰でも使える、シンプルで軽量なテキストエディタを開発しています。Vim や Emacs のような学習コストはなく、かつメモ帳のような機能不足でもない。ちょうど良いバランスのエディタを目指しています。

## 主な特徴

- **軽量**: 起動が速く、メモリ使用量も少ない。古いPCでも快適に動作します。
- **プラグインで拡張**: 必要に応じて機能を追加できるプラグインAPIを提供。LSP（Language Server Protocol）対応で、コード補完やリファクタリングも可能です。
- **クロスプラットフォーム**: Windows / macOS / Linux で同じ操作性を実現。
- **オープンソース**: MIT ライセンスで、誰でも自由に利用・改変・再配布できます。

## 開発の背景

現代のエディタは高機能になりすぎて、シンプルにテキストを編集したいだけのユーザーには重いと感じることがあります。一方で、メモ帳では物足りない。その中間を埋めるエディタが欲しいと考え、このプロジェクトを始めました。

## 今後の予定

- マルチカーソル・マルチ選択の強化
- 検索・置換の改善（正規表現、ファイル検索）
- テーマシステムの拡張
- モバイル版の検討

ご支援いただいた方には、リリースノートや開発の近況を定期的にお届けします。よろしくお願いいたします。`,
    status: "active",
    owner_want_monthly: 50000,
    created_at: day(2),
    updated_at: day(1),
    cost_items: [
      { label: "サーバー費用", unit_price: 3000, quantity: 1 },
      { label: "開発費", unit_price: 15000, quantity: 2 },
      { label: "その他", unit_price: 2000, quantity: 1 },
    ],
    monthly_target: 35000,
    alerts: { warning_threshold: 50, critical_threshold: 20 },
    _mockCurrentMonthly: 42000, // 84%
  },
  {
    id: "mock-2",
    owner_id: "user-2",
    name: "無料の日本語学習アプリ",
    description: "日本語を学びたい人のための、完全無料の学習アプリ。広告なし。",
    _mockImageUrl:
      "https://placehold.co/800x400/6b9b6e/ffffff?text=Japanese+App",
    _mockOverview: `日本語を学びたい人のための、完全無料の学習アプリです。広告なし、課金なし。純粋に学びに集中できる環境を提供します。

JLPT（日本語能力試験）N5〜N1 に対応したコンテンツを順次追加しています。単語、文法、聴解、読解の各セクションがあり、スキマ時間で学習できます。`,
    status: "active",
    owner_want_monthly: 80000,
    created_at: day(30),
    updated_at: day(28),
    cost_items: [
      { label: "サーバー費用", unit_price: 5000, quantity: 1 },
      { label: "開発費", unit_price: 12000, quantity: 4 },
      { label: "その他", unit_price: 3000, quantity: 1 },
    ],
    monthly_target: 56000,
    alerts: { warning_threshold: 50, critical_threshold: 20 },
    _mockCurrentMonthly: 95000, // 119% 超達成
  },
  {
    id: "mock-3",
    owner_id: "user-3",
    name: "アクセシビリティチェックツール",
    description: "Webサイトのアクセシビリティを無料でチェック。WCAG 2.1 対応。",
    _mockImageUrl: "https://placehold.co/800x400/6b9b6e/ffffff?text=A11y+Tool",
    status: "frozen",
    owner_want_monthly: 30000,
    created_at: day(5),
    updated_at: day(4),
    cost_items: [
      { label: "サーバー費用", unit_price: 2000, quantity: 1 },
      { label: "開発費", unit_price: 10000, quantity: 2 },
      { label: "その他", unit_price: 1000, quantity: 1 },
    ],
    monthly_target: 23000,
    alerts: { warning_threshold: 50, critical_threshold: 20 },
    _mockCurrentMonthly: 8000, // 27% 危険
  },
  {
    id: "mock-4",
    owner_id: "user-1",
    name: "GIVErS プラットフォーム",
    description: "このプラットフォーム自体。GIVEの精神で運営。手数料ゼロ。",
    _mockImageUrl: "https://placehold.co/800x400/4a7c59/ffffff?text=GIVErS",
    _mockOverview: `## GIVErS とは

GIVErS は「GIVE の精神」に基づく寄付プラットフォームです。作り手の GIVE（見返りを求めず良いものを作る）と、受け手の GIVE（使って良かった人が自発的に応援する）をつなぎます。

## プラットフォームの特徴

- **手数料ゼロ**: 寄付金から手数料を徴収しません。Stripe の決済手数料のみが発生します。
- **透明性**: プロジェクトごとに月額目標・達成率・コスト内訳を公開。寄付者が判断しやすい情報を提供します。
- **軽量SNS寄り**: アクティビティフィードで「いま起きていること」を感じられる設計を目指しています。

## 運営方針

GIVErS 自体も GIVE で運営されます。まわらなくなったら、世に受け入れられなかったということ。原理を曲げてまで存続はしません。ある日突然クローズする可能性もあるため、透明性を保ちます。

## 資金の使途

- サーバー・インフラ費用
- 開発・保守のための時間確保
- その他運営経費

ご支援はプラットフォームの継続と改善に充てさせていただきます。`,
    status: "active",
    owner_want_monthly: 100000,
    created_at: day(90),
    updated_at: day(1),
    cost_items: [
      { label: "サーバー・インフラ", unit_price: 10000, quantity: 1 },
      { label: "開発・保守", unit_price: 15000, quantity: 4 },
      { label: "その他運営経費", unit_price: 5000, quantity: 1 },
    ],
    monthly_target: 75000,
    alerts: { warning_threshold: 50, critical_threshold: 20 },
    _mockCurrentMonthly: 52000, // 52% 注意
  },
  {
    id: "mock-5",
    owner_id: "user-4",
    name: "子ども向けプログラミング教材",
    description: "小学校で使える、無料のプログラミング教材。Scratch ベース。",
    _mockImageUrl: "https://placehold.co/800x400/6b9b6e/ffffff?text=Kids+Code",
    status: "active",
    owner_want_monthly: 40000,
    created_at: day(1),
    updated_at: day(0),
    cost_items: [
      { label: "サーバー費用", unit_price: 2000, quantity: 1 },
      { label: "開発費", unit_price: 8000, quantity: 3 },
      { label: "その他", unit_price: 2000, quantity: 1 },
    ],
    monthly_target: 28000,
    alerts: { warning_threshold: 50, critical_threshold: 20 },
    _mockCurrentMonthly: 20000, // 50% 境界
  },
  {
    id: "mock-6",
    owner_id: "user-2",
    name: "翻訳ボランティア支援サイト",
    description:
      "オープンソースプロジェクトの翻訳を支援。翻訳者とプロジェクトをつなぐ。",
    _mockImageUrl:
      "https://placehold.co/800x400/6b9b6e/ffffff?text=Translation",
    status: "active",
    owner_want_monthly: 25000,
    created_at: day(14),
    updated_at: day(10),
    cost_items: [
      { label: "サーバー費用", unit_price: 3000, quantity: 1 },
      { label: "開発費", unit_price: 10000, quantity: 1 },
      { label: "その他", unit_price: 2000, quantity: 1 },
    ],
    monthly_target: 15000,
    alerts: { warning_threshold: 50, critical_threshold: 20 },
    _mockCurrentMonthly: 22000, // 88%
  },
];
