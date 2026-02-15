/** アクティビティフィード用のモックデータ */

export type ActivityType =
  | 'project_created'
  | 'project_updated'
  | 'donation'
  | 'milestone';

export interface ActivityItem {
  id: string;
  type: ActivityType;
  created_at: string;
  project_id: string;
  project_name: string;
  /** アクター（匿名の場合は null） */
  actor_name: string | null;
  /** 寄付額（donation 時） */
  amount?: number;
  /** マイルストーン達成率（milestone 時、例: 50） */
  rate?: number;
}

const now = new Date();
const time = (d: number, h = 12) => {
  const t = new Date(now);
  t.setDate(t.getDate() - d);
  t.setHours(h, 0, 0, 0);
  return t.toISOString();
};

export const MOCK_OWNERS: Record<string, string> = {
  'user-1': '山田太郎',
  'user-2': '佐藤花子',
  'user-3': '鈴木一郎',
  'user-4': '田中美咲',
};

/** プロジェクト別の最近の応援者（モック） */
export const MOCK_RECENT_SUPPORTERS: Record<string, { name: string | null; amount: number }[]> = {
  'mock-1': [
    { name: '佐藤花子', amount: 3000 },
    { name: null, amount: 1500 },
    { name: '鈴木一郎', amount: 1000 },
  ],
  'mock-2': [
    { name: null, amount: 5000 },
    { name: '山田太郎', amount: 2000 },
  ],
  'mock-3': [
    { name: '鈴木一郎', amount: 500 },
    { name: null, amount: 300 },
  ],
  'mock-4': [
    { name: '佐藤花子', amount: 5000 },
    { name: null, amount: 2000 },
  ],
  'mock-5': [
    { name: null, amount: 1000 },
  ],
  'mock-6': [
    { name: '山田太郎', amount: 2000 },
    { name: null, amount: 1000 },
  ],
};

export const MOCK_ACTIVITIES: ActivityItem[] = [
  {
    id: 'act-1',
    type: 'donation',
    created_at: time(0, 9),
    project_id: 'mock-5',
    project_name: '子ども向けプログラミング教材',
    actor_name: null,
    amount: 1000,
  },
  {
    id: 'act-2',
    type: 'project_created',
    created_at: time(0, 8),
    project_id: 'mock-5',
    project_name: '子ども向けプログラミング教材',
    actor_name: '田中美咲',
  },
  {
    id: 'act-3',
    type: 'donation',
    created_at: time(1, 14),
    project_id: 'mock-1',
    project_name: 'オープンソースの軽量エディタ',
    actor_name: '佐藤花子',
    amount: 3000,
  },
  {
    id: 'act-4',
    type: 'milestone',
    created_at: time(1, 10),
    project_id: 'mock-6',
    project_name: '翻訳ボランティア支援サイト',
    actor_name: null,
    rate: 88,
  },
  {
    id: 'act-5',
    type: 'donation',
    created_at: time(2, 16),
    project_id: 'mock-2',
    project_name: '無料の日本語学習アプリ',
    actor_name: null,
    amount: 5000,
  },
  {
    id: 'act-6',
    type: 'project_updated',
    created_at: time(2, 11),
    project_id: 'mock-4',
    project_name: 'GIVErS プラットフォーム',
    actor_name: '山田太郎',
  },
  {
    id: 'act-7',
    type: 'milestone',
    created_at: time(3, 9),
    project_id: 'mock-1',
    project_name: 'オープンソースの軽量エディタ',
    actor_name: null,
    rate: 80,
  },
  {
    id: 'act-8',
    type: 'donation',
    created_at: time(4, 13),
    project_id: 'mock-3',
    project_name: 'アクセシビリティチェックツール',
    actor_name: '鈴木一郎',
    amount: 500,
  },
];
