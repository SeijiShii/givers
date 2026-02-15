import type { Project } from '../lib/api';

const now = new Date();
const day = (d: number) => {
  const d2 = new Date(now);
  d2.setDate(d2.getDate() - d);
  return d2.toISOString();
};

/** 達成率を計算するための currentMonthly をモックで持つ */
export interface MockProject extends Project {
  _mockCurrentMonthly?: number;
}

export const MOCK_PROJECTS: MockProject[] = [
  {
    id: 'mock-1',
    owner_id: 'user-1',
    name: 'オープンソースの軽量エディタ',
    description: '誰でも使える、シンプルで軽量なテキストエディタ。プラグインで拡張可能。',
    status: 'active',
    owner_want_monthly: 50000,
    created_at: day(2),
    updated_at: day(1),
    costs: {
      server_cost_monthly: 3000,
      dev_cost_per_day: 15000,
      dev_days_per_month: 2,
      other_cost_monthly: 2000,
    },
    alerts: { warning_threshold: 50, critical_threshold: 20 },
    _mockCurrentMonthly: 42000, // 84%
  },
  {
    id: 'mock-2',
    owner_id: 'user-2',
    name: '無料の日本語学習アプリ',
    description: '日本語を学びたい人のための、完全無料の学習アプリ。広告なし。',
    status: 'active',
    owner_want_monthly: 80000,
    created_at: day(30),
    updated_at: day(28),
    costs: {
      server_cost_monthly: 5000,
      dev_cost_per_day: 12000,
      dev_days_per_month: 4,
      other_cost_monthly: 3000,
    },
    alerts: { warning_threshold: 50, critical_threshold: 20 },
    _mockCurrentMonthly: 95000, // 119% 超達成
  },
  {
    id: 'mock-3',
    owner_id: 'user-3',
    name: 'アクセシビリティチェックツール',
    description: 'Webサイトのアクセシビリティを無料でチェック。WCAG 2.1 対応。',
    status: 'active',
    owner_want_monthly: 30000,
    created_at: day(5),
    updated_at: day(4),
    costs: {
      server_cost_monthly: 2000,
      dev_cost_per_day: 10000,
      dev_days_per_month: 2,
      other_cost_monthly: 1000,
    },
    alerts: { warning_threshold: 50, critical_threshold: 20 },
    _mockCurrentMonthly: 8000, // 27% 危険
  },
  {
    id: 'mock-4',
    owner_id: 'user-1',
    name: 'GIVErS プラットフォーム',
    description: 'このプラットフォーム自体。GIVEの精神で運営。手数料ゼロ。',
    status: 'active',
    owner_want_monthly: 100000,
    created_at: day(90),
    updated_at: day(1),
    costs: {
      server_cost_monthly: 10000,
      dev_cost_per_day: 15000,
      dev_days_per_month: 4,
      other_cost_monthly: 5000,
    },
    alerts: { warning_threshold: 50, critical_threshold: 20 },
    _mockCurrentMonthly: 52000, // 52% 注意
  },
  {
    id: 'mock-5',
    owner_id: 'user-4',
    name: '子ども向けプログラミング教材',
    description: '小学校で使える、無料のプログラミング教材。Scratch ベース。',
    status: 'active',
    owner_want_monthly: 40000,
    created_at: day(1),
    updated_at: day(0),
    costs: {
      server_cost_monthly: 2000,
      dev_cost_per_day: 8000,
      dev_days_per_month: 3,
      other_cost_monthly: 2000,
    },
    alerts: { warning_threshold: 50, critical_threshold: 20 },
    _mockCurrentMonthly: 20000, // 50% 境界
  },
  {
    id: 'mock-6',
    owner_id: 'user-2',
    name: '翻訳ボランティア支援サイト',
    description: 'オープンソースプロジェクトの翻訳を支援。翻訳者とプロジェクトをつなぐ。',
    status: 'active',
    owner_want_monthly: 25000,
    created_at: day(14),
    updated_at: day(10),
    costs: {
      server_cost_monthly: 3000,
      dev_cost_per_day: 10000,
      dev_days_per_month: 1,
      other_cost_monthly: 2000,
    },
    alerts: { warning_threshold: 50, critical_threshold: 20 },
    _mockCurrentMonthly: 22000, // 88%
  },
];
