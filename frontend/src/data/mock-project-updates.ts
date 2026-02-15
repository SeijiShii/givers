import type { ProjectUpdate } from '../lib/api';

const now = new Date();
const time = (d: number, h = 12) => {
  const t = new Date(now);
  t.setDate(t.getDate() - d);
  t.setHours(h, 0, 0, 0);
  return t.toISOString();
};

/** プロジェクト別のオーナーアップデート（モック） */
export const MOCK_PROJECT_UPDATES: Record<string, ProjectUpdate[]> = {
  'mock-1': [
    {
      id: 'up-1-1',
      project_id: 'mock-1',
      created_at: time(0, 10),
      title: 'v0.8.0 リリースしました',
      body: 'プラグインAPIの安定化と、新しいキーバインド設定UIを追加しました。多くのフィードバックをいただきありがとうございます。次は検索機能の強化に取り組みます。',
      author_name: '山田太郎',
    },
    {
      id: 'up-1-2',
      project_id: 'mock-1',
      created_at: time(7, 14),
      title: '支援状況のご報告',
      body: '今月は目標の84%まで到達しました。継続的にご支援いただいている皆様、本当にありがとうございます。来月はマルチカーソル機能の実装に着手する予定です。',
      author_name: '山田太郎',
    },
    {
      id: 'up-1-3',
      project_id: 'mock-1',
      created_at: time(14, 9),
      body: 'GitHubのスターが1000を超えました！コミュニティの皆様のおかげです。引き続きよろしくお願いします。',
      author_name: '山田太郎',
    },
  ],
  'mock-4': [
    {
      id: 'up-4-1',
      project_id: 'mock-4',
      created_at: time(0, 11),
      title: 'Stripe Connect の検討状況',
      body: '決済連携について、Stripe Connect Standard プランを採用する方向で検討を進めています。プロジェクトオーナーが自分の Stripe アカウントに直接入金できる形を目指します。プラットフォーム側の追加コストはゼロになる見込みです。',
      author_name: '山田太郎',
    },
    {
      id: 'up-4-2',
      project_id: 'mock-4',
      created_at: time(5, 15),
      title: 'UX モックの進捗',
      body: 'プロジェクト詳細ページにタブUI（支援状況・概要・アップデート）を追加しました。寄付フォームやチャート表示も含め、モックで体験を検証できる状態になってきました。',
      author_name: '山田太郎',
    },
  ],
  'mock-2': [
    {
      id: 'up-2-1',
      project_id: 'mock-2',
      created_at: time(3, 10),
      title: 'N5 レベル完成',
      body: '日本語能力試験N5レベルのコンテンツが一通り揃いました。次はN4レベルに着手します。',
      author_name: '佐藤花子',
    },
  ],
};
