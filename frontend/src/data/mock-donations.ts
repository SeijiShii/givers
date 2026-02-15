/** 寄付履歴（モック） */
export interface MockDonation {
  id: string;
  user_id: string;
  project_id: string;
  project_name: string;
  amount: number;
  created_at: string;
  message?: string | null;
}

/** 定期寄付（モック） */
export interface MockRecurringDonation {
  id: string;
  user_id: string;
  project_id: string;
  project_name: string;
  amount: number;
  created_at: string;
  status: 'active' | 'paused' | 'cancelled';
  interval?: 'monthly' | 'yearly';
}

const now = new Date();
const time = (d: number, h = 12) => {
  const t = new Date(now);
  t.setDate(t.getDate() - d);
  t.setHours(h, 0, 0, 0);
  return t.toISOString();
};

/** ユーザー別の寄付履歴（user-2 メンバー用） */
export const MOCK_DONATIONS: Record<string, MockDonation[]> = {
  'user-2': [
    { id: 'd-1', user_id: 'user-2', project_id: 'mock-1', project_name: 'オープンソースの軽量エディタ', amount: 3000, created_at: time(1, 14), message: '応援しています！' },
    { id: 'd-2', user_id: 'user-2', project_id: 'mock-4', project_name: 'GIVErS プラットフォーム', amount: 2000, created_at: time(5, 10) },
    { id: 'd-3', user_id: 'user-2', project_id: 'mock-1', project_name: 'オープンソースの軽量エディタ', amount: 1000, created_at: time(14, 9) },
  ],
  'user-1': [
    { id: 'd-4', user_id: 'user-1', project_id: 'mock-2', project_name: '無料の日本語学習アプリ', amount: 5000, created_at: time(2, 16) },
    { id: 'd-5', user_id: 'user-1', project_id: 'mock-6', project_name: '翻訳ボランティア支援サイト', amount: 2000, created_at: time(10, 11) },
  ],
  'user-6': [
    { id: 'd-6', user_id: 'user-6', project_id: 'mock-1', project_name: 'オープンソースの軽量エディタ', amount: 2000, created_at: time(3, 11) },
    { id: 'd-7', user_id: 'user-6', project_id: 'mock-2', project_name: '無料の日本語学習アプリ', amount: 1000, created_at: time(7, 9) },
  ],
};

/** ユーザー別の定期寄付（user-2 メンバー用） */
export const MOCK_RECURRING_DONATIONS: Record<string, MockRecurringDonation[]> = {
  'user-2': [
    { id: 'r-1', user_id: 'user-2', project_id: 'mock-1', project_name: 'オープンソースの軽量エディタ', amount: 1000, created_at: time(30), status: 'active', interval: 'monthly' },
    { id: 'r-2', user_id: 'user-2', project_id: 'mock-4', project_name: 'GIVErS プラットフォーム', amount: 500, created_at: time(60), status: 'active', interval: 'monthly' },
  ],
  'user-1': [
    { id: 'r-3', user_id: 'user-1', project_id: 'mock-2', project_name: '無料の日本語学習アプリ', amount: 2000, created_at: time(45), status: 'active', interval: 'monthly' },
  ],
  'user-6': [
    { id: 'r-4', user_id: 'user-6', project_id: 'mock-1', project_name: 'オープンソースの軽量エディタ', amount: 500, created_at: time(20), status: 'active', interval: 'monthly' },
  ],
};
