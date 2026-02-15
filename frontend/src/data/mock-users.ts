import type { User, AdminUser } from '../lib/api';

const now = new Date();
const iso = (d: number) => {
  const t = new Date(now);
  t.setDate(t.getDate() - d);
  return t.toISOString();
};

/** モック用ユーザー（role 付き） */
export interface MockUser extends User {
  role: 'host' | 'project_owner' | 'donor';
}

/** ホストユーザー（user-1） */
export const MOCK_HOST_USER: MockUser = {
  id: 'user-1',
  email: 'host@givers.example',
  name: '山田太郎',
  created_at: iso(365),
  updated_at: iso(1),
  role: 'host',
};

/** メンバーユーザー（プロジェクトオーナー） */
export const MOCK_MEMBER_USER: MockUser = {
  id: 'user-2',
  email: 'member@givers.example',
  name: '佐藤花子',
  created_at: iso(180),
  updated_at: iso(2),
  role: 'project_owner',
};

/** 寄付メンバー（寄付者のみ、プロジェクトなし） */
export const MOCK_DONOR_USER: MockUser = {
  id: 'user-6',
  email: 'donor@givers.example',
  name: '高橋健太',
  created_at: iso(90),
  updated_at: iso(3),
  role: 'donor',
};

/** ユーザー管理画面用のユーザー一覧 */
export const MOCK_ADMIN_USERS: AdminUser[] = [
  { ...MOCK_HOST_USER, status: 'active', project_count: 2, role: 'host' },
  { ...MOCK_MEMBER_USER, status: 'active', project_count: 1, role: 'project_owner' },
  {
    id: 'user-3',
    email: 'suzuki@givers.example',
    name: '鈴木一郎',
    created_at: iso(120),
    updated_at: iso(5),
    status: 'active',
    project_count: 1,
  },
  {
    id: 'user-4',
    email: 'tanaka@givers.example',
    name: '田中美咲',
    created_at: iso(90),
    updated_at: iso(3),
    status: 'active',
    project_count: 1,
  },
  {
    id: 'user-5',
    email: 'suspended@givers.example',
    name: '停止中ユーザー',
    created_at: iso(60),
    updated_at: iso(30),
    status: 'suspended',
    project_count: 0,
  },
  { ...MOCK_DONOR_USER, status: 'active', project_count: 0 },
];
