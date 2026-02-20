import type {
  Project,
  ProjectCosts,
  ProjectAlerts,
  CreateProjectInput,
  UpdateProjectInput,
  User,
  Donation,
  RecurringDonation,
} from './api';
import type { AdminUser, DisclosureExportPayload } from './api';
import { MOCK_LOGIN_MODE_KEY } from './api';

/** モック: トークン→アカウント移行済みフラグ（localStorage）。true なら getMe で pending_token_migration を返さない */
const MOCK_MIGRATION_DONE_KEY = 'givers_mock_migration_done';
/** モック: 利用停止ユーザーをシミュレート（localStorage）。true なら getMe で suspended: true を返す */
const MOCK_SUSPENDED_USER_KEY = 'givers_mock_suspended_user';
import { MOCK_HOST_USER, MOCK_MEMBER_USER, MOCK_DONOR_USER, MOCK_ADMIN_USERS } from '../data/mock-users';
import type { ProjectUpdate, CreateProjectUpdateInput } from './api';
import { MOCK_PROJECTS, type MockProject } from '../data/mock-projects';
import { MOCK_ACTIVITIES, MOCK_OWNERS, MOCK_RECENT_SUPPORTERS, type ActivityItem } from '../data/mock-activities';
import { MOCK_CHART_DATA, type ChartDataPoint } from '../data/mock-chart-data';
import { MOCK_PROJECT_UPDATES } from '../data/mock-project-updates';
import { MOCK_DONATIONS, MOCK_RECURRING_DONATIONS } from '../data/mock-donations';

const delay = (ms: number) => new Promise((r) => setTimeout(r, ms));

/** モック用の擬似遅延（体感用、0 にしても可） */
const MOCK_DELAY = 150;

/** 定期寄付キャンセル済み ID（セッション内のみ、モック用） */
const cancelledRecurringIds = new Set<string>();

/** 定期寄付一時休止 ID（セッション内のみ、モック用） */
const pausedRecurringIds = new Set<string>();

/** 定期寄付削除済み ID（セッション内のみ、モック用。一覧から非表示） */
const deletedRecurringIds = new Set<string>();

/** 定期寄付の変更（金額・タイミング）上書き（セッション内のみ、モック用） */
const recurringOverrides = new Map<string, { amount?: number; interval?: 'monthly' | 'yearly' }>();

/** プロジェクトステータス上書き（セッション内のみ、モック用） */
const projectStatusOverrides = new Map<string, string>();

/** プロジェクト概要上書き（セッション内のみ、モック用） */
const projectOverviewOverrides = new Map<string, string>();

/** プロジェクトアップデート（初期値 + セッション内投稿、モック用） */
const projectUpdatesStore = new Map<string, ProjectUpdate[]>();

function getProjectUpdatesList(projectId: string): ProjectUpdate[] {
  if (!projectUpdatesStore.has(projectId)) {
    projectUpdatesStore.set(projectId, [...(MOCK_PROJECT_UPDATES[projectId] ?? [])]);
  }
  return projectUpdatesStore.get(projectId)!;
}

function toProject(p: MockProject): Project {
  const { _mockCurrentMonthly, _mockImageUrl, _mockOverview, ...rest } = p;
  return {
    ...rest,
    current_monthly_donations: _mockCurrentMonthly,
    owner_name: MOCK_OWNERS[p.owner_id],
    recent_supporters: MOCK_RECENT_SUPPORTERS[p.id] ?? [],
    image_url: _mockImageUrl ?? null,
    overview: _mockOverview ?? rest.description,
  };
}

export const mockApi = {
  async healthCheck(): Promise<{ status: string; message: string }> {
    await delay(MOCK_DELAY);
    return { status: 'ok', message: 'GIVErS API (Mock)' };
  },

  async getMe(): Promise<User | null> {
    await delay(MOCK_DELAY);
    if (typeof window === 'undefined' || !window.localStorage) {
      return MOCK_HOST_USER;
    }
    const mode = window.localStorage.getItem(MOCK_LOGIN_MODE_KEY);
    if (mode === 'logout') return null;
    if (mode === 'donor') {
      const migrationDone = window.localStorage.getItem(MOCK_MIGRATION_DONE_KEY) === 'true';
      const suspended = window.localStorage.getItem(MOCK_SUSPENDED_USER_KEY) === 'true';
      return {
        ...MOCK_DONOR_USER,
        pending_token_migration: migrationDone ? undefined : true,
        ...(suspended ? { suspended: true as const } : {}),
      };
    }
    if (mode === 'project_owner' || mode === 'member') {
      const suspended = window.localStorage.getItem(MOCK_SUSPENDED_USER_KEY) === 'true';
      return { ...MOCK_MEMBER_USER, ...(suspended ? { suspended: true as const } : {}) };
    }
    const suspended = window.localStorage.getItem(MOCK_SUSPENDED_USER_KEY) === 'true';
    return { ...MOCK_HOST_USER, ...(suspended ? { suspended: true as const } : {}) };
  },

  async migrateFromToken(): Promise<{ migrated_count: number; already_migrated?: boolean }> {
    await delay(MOCK_DELAY);
    if (typeof window !== 'undefined' && window.localStorage) {
      if (window.localStorage.getItem(MOCK_MIGRATION_DONE_KEY) === 'true') {
        return { migrated_count: 0, already_migrated: true };
      }
      window.localStorage.setItem(MOCK_MIGRATION_DONE_KEY, 'true');
    }
    return { migrated_count: 1, already_migrated: false };
  },

  async getGoogleLoginUrl(): Promise<{ url: string }> {
    await delay(MOCK_DELAY);
    return { url: '#' };
  },

  async getGitHubLoginUrl(): Promise<{ url: string }> {
    await delay(MOCK_DELAY);
    return { url: '#' };
  },

  async getAppleLoginUrl(): Promise<{ url: string }> {
    await delay(MOCK_DELAY);
    return { url: '#' };
  },

  async getEmailLoginUrl(): Promise<{ url: string }> {
    await delay(MOCK_DELAY);
    return { url: '#' };
  },

  async logout(): Promise<void> {
    await delay(MOCK_DELAY);
  },

  async getProjects(limit = 20, offset = 0): Promise<Project[]> {
    await delay(MOCK_DELAY);
    const list = MOCK_PROJECTS.slice(offset, offset + limit).map(toProject);
    return list;
  },

  async getProject(id: string): Promise<Project> {
    await delay(MOCK_DELAY);
    const p = MOCK_PROJECTS.find((x) => x.id === id);
    if (!p) throw new Error('Project not found');
    const overridden = projectStatusOverrides.get(id);
    const merged = overridden ? { ...p, status: overridden } : p;
    const project = toProject(merged);
    const overviewOverride = projectOverviewOverrides.get(id);
    if (overviewOverride != null) project.overview = overviewOverride;
    return project;
  },

  async getMyProjects(): Promise<Project[]> {
    await delay(MOCK_DELAY);
    const me = await this.getMe();
    if (!me) return [];
    const list = MOCK_PROJECTS.filter((p) => p.owner_id === me.id).map((p) => {
      const overridden = projectStatusOverrides.get(p.id);
      return overridden ? { ...p, status: overridden } : p;
    });
    return list.map(toProject);
  },

  async getMyDonations(): Promise<Donation[]> {
    await delay(MOCK_DELAY);
    const me = await this.getMe();
    if (!me) return [];
    const list = MOCK_DONATIONS[me.id] ?? [];
    return [...list].sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
  },

  async getMyRecurringDonations(): Promise<RecurringDonation[]> {
    await delay(MOCK_DELAY);
    const me = await this.getMe();
    if (!me) return [];
    const list = (MOCK_RECURRING_DONATIONS[me.id] ?? [])
      .filter((r) => !deletedRecurringIds.has(r.id))
      .map((r) => {
        const overrides = recurringOverrides.get(r.id);
        let status = r.status;
        if (cancelledRecurringIds.has(r.id)) status = 'cancelled';
        else if (pausedRecurringIds.has(r.id)) status = 'paused';
        return {
          ...r,
          ...overrides,
          status,
          interval: overrides?.interval ?? r.interval ?? 'monthly',
        };
      });
    return list;
  },

  async cancelRecurringDonation(id: string): Promise<void> {
    await delay(MOCK_DELAY);
    cancelledRecurringIds.add(id);
  },

  async updateRecurringDonation(
    id: string,
    input: { amount?: number; interval?: 'monthly' | 'yearly' }
  ): Promise<RecurringDonation> {
    await delay(MOCK_DELAY);
    const me = await this.getMe();
    if (!me) throw new Error('Not logged in');
    const list = (MOCK_RECURRING_DONATIONS[me.id] ?? []).filter((r) => !deletedRecurringIds.has(r.id));
    const r = list.find((x) => x.id === id);
    if (!r) throw new Error('Recurring donation not found');
    const current = recurringOverrides.get(id) ?? {};
    recurringOverrides.set(id, { ...current, ...input });
    const all = await this.getMyRecurringDonations();
    const updated = all.find((u) => u.id === id);
    if (!updated) throw new Error('Recurring donation not found');
    return updated;
  },

  async pauseRecurringDonation(id: string): Promise<void> {
    await delay(MOCK_DELAY);
    pausedRecurringIds.add(id);
  },

  async resumeRecurringDonation(id: string): Promise<void> {
    await delay(MOCK_DELAY);
    pausedRecurringIds.delete(id);
  },

  async deleteRecurringDonation(id: string): Promise<void> {
    await delay(MOCK_DELAY);
    deletedRecurringIds.add(id);
  },

  async createProject(input: CreateProjectInput): Promise<Project> {
    await delay(MOCK_DELAY);
    const me = await this.getMe();
    const ownerId = me?.id ?? 'user-mock';
    const id = `mock-new-${Date.now()}`;
    const now = new Date().toISOString();
    const newProject: Project = {
      id,
      owner_id: ownerId,
      name: input.name,
      description: input.description ?? '',
      status: input.status ?? 'active',
      owner_want_monthly: input.owner_want_monthly ?? null,
      created_at: now,
      updated_at: now,
      costs: input.costs ?? null,
      alerts: input.alerts ?? null,
    };
    return newProject;
  },

  async updateProject(id: string, input: UpdateProjectInput): Promise<Project> {
    await delay(MOCK_DELAY);
    const p = MOCK_PROJECTS.find((x) => x.id === id);
    if (!p) throw new Error('Project not found');
    if (input.status != null) projectStatusOverrides.set(id, input.status);
    if (input.overview != null) projectOverviewOverrides.set(id, input.overview);
    const merged = {
      ...p,
      ...input,
      id: p.id,
      owner_id: p.owner_id,
      created_at: p.created_at,
      status: input.status ?? projectStatusOverrides.get(id) ?? p.status,
    };
    const project = toProject(merged);
    const overviewOverride = projectOverviewOverrides.get(id);
    if (overviewOverride != null) project.overview = overviewOverride;
    return project;
  },

  /** 新着プロジェクト（created_at 降順） */
  async getNewProjects(limit = 5): Promise<Project[]> {
    await delay(MOCK_DELAY);
    const sorted = [...MOCK_PROJECTS].sort(
      (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    );
    return sorted.slice(0, limit).map(toProject);
  },

  /** HOT プロジェクト（達成率・人気でソート） */
  async getHotProjects(limit = 5): Promise<Project[]> {
    await delay(MOCK_DELAY);
    const withRate = MOCK_PROJECTS.map((p) => {
      const target = p.owner_want_monthly ?? 0;
      const current = p._mockCurrentMonthly ?? 0;
      const rate = target > 0 ? (current / target) * 100 : 0;
      return { p, rate };
    });
    const sorted = withRate.sort((a, b) => b.rate - a.rate);
    return sorted.slice(0, limit).map((x) => toProject(x.p));
  },

  /** 関連プロジェクト（当該を除く HOT 順で最大 limit 件） */
  async getRelatedProjects(projectId: string, limit = 4): Promise<Project[]> {
    await delay(MOCK_DELAY);
    const hot = await this.getHotProjects(limit + MOCK_PROJECTS.length);
    return hot.filter((p) => p.id !== projectId).slice(0, limit);
  },

  /** アクティビティフィード */
  async getActivityFeed(limit = 10): Promise<ActivityItem[]> {
    await delay(MOCK_DELAY);
    const sorted = [...MOCK_ACTIVITIES].sort(
      (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    );
    return sorted.slice(0, limit);
  },

  /** プロジェクトチャートデータ */
  async getProjectChart(projectId: string): Promise<ChartDataPoint[]> {
    await delay(MOCK_DELAY);
    return MOCK_CHART_DATA[projectId] ?? [];
  },

  /** ユーザー一覧（ホスト用） */
  async getAdminUsers(): Promise<AdminUser[]> {
    await delay(MOCK_DELAY);
    return [...MOCK_ADMIN_USERS];
  },

  /** 開示用データ出力（ホスト用。第三者情報開示請求等に備える） */
  async getDisclosureExport(type: 'user' | 'project', id: string): Promise<DisclosureExportPayload> {
    await delay(MOCK_DELAY);
    const exported_at = new Date().toISOString();
    const platform = 'GIVErS';

    if (type === 'user') {
      const user = MOCK_ADMIN_USERS.find((u) => u.id === id);
      if (!user) throw new Error('User not found');
      const user_projects = MOCK_PROJECTS.filter((p) => p.owner_id === id).map((p) => ({
        id: p.id,
        name: p.name,
        status: p.status,
        created_at: p.created_at,
      }));
      const user_donations = (MOCK_DONATIONS[id] ?? []).map((d) => ({
        id: d.id,
        project_id: d.project_id,
        project_name: d.project_name,
        amount: d.amount,
        created_at: d.created_at,
      }));
      const user_recurring = (MOCK_RECURRING_DONATIONS[id] ?? []).map((r) => ({
        id: r.id,
        project_id: r.project_id,
        project_name: r.project_name,
        amount: r.amount,
        created_at: r.created_at,
        status: r.status,
        interval: r.interval,
      }));
      return {
        exported_at,
        platform,
        type: 'user',
        user: {
          id: user.id,
          email: user.email,
          name: user.name,
          created_at: user.created_at,
          updated_at: user.updated_at,
          status: user.status,
          role: user.role,
        },
        user_projects,
        user_donations,
        user_recurring,
      };
    }

    const p = MOCK_PROJECTS.find((x) => x.id === id);
    if (!p) throw new Error('Project not found');
    const owner = MOCK_ADMIN_USERS.find((u) => u.id === p.owner_id);
    const allDonations = Object.values(MOCK_DONATIONS).flat();
    const projectDonations = allDonations.filter((d) => d.project_id === id).map((d) => ({
      id: d.id,
      amount: d.amount,
      created_at: d.created_at,
      donor_type: 'user' as const,
    }));
    return {
      exported_at,
      platform,
      type: 'project',
      project: {
        id: p.id,
        name: p.name,
        description: p.description,
        owner_id: p.owner_id,
        status: p.status,
        created_at: p.created_at,
        owner_name: MOCK_OWNERS[p.owner_id],
      },
      project_donations: projectDonations,
    };
  },

  /** プロジェクトオーナーからのアップデート */
  async getProjectUpdates(projectId: string, limit = 20): Promise<ProjectUpdate[]> {
    await delay(MOCK_DELAY);
    const list = getProjectUpdatesList(projectId);
    return list
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
      .slice(0, limit);
  },

  /** アップデート投稿（オーナー限定） */
  async createProjectUpdate(projectId: string, input: CreateProjectUpdateInput): Promise<ProjectUpdate> {
    await delay(MOCK_DELAY);
    const me = await this.getMe();
    const p = MOCK_PROJECTS.find((x) => x.id === projectId);
    if (!p) throw new Error('Project not found');
    if (!me || p.owner_id !== me.id) throw new Error('Only project owner can post updates');
    const list = getProjectUpdatesList(projectId);
    const now = new Date().toISOString();
    const newUpdate: ProjectUpdate = {
      id: `up-${projectId}-${Date.now()}`,
      project_id: projectId,
      created_at: now,
      title: input.title ?? null,
      body: input.body,
      author_name: me.name,
    };
    list.unshift(newUpdate);
    return newUpdate;
  },

  /** アップデート編集（オーナー限定） */
  async updateProjectUpdate(
    projectId: string,
    updateId: string,
    input: { title?: string | null; body?: string }
  ): Promise<ProjectUpdate> {
    await delay(MOCK_DELAY);
    const me = await this.getMe();
    const p = MOCK_PROJECTS.find((x) => x.id === projectId);
    if (!p) throw new Error('Project not found');
    if (!me || p.owner_id !== me.id) throw new Error('Only project owner can edit updates');
    const list = getProjectUpdatesList(projectId);
    const idx = list.findIndex((u) => u.id === updateId);
    if (idx < 0) throw new Error('Update not found');
    const updated = { ...list[idx], ...input };
    list[idx] = updated;
    return updated;
  },

  /** アップデート削除（オーナー限定） */
  async deleteProjectUpdate(projectId: string, updateId: string): Promise<void> {
    await delay(MOCK_DELAY);
    const me = await this.getMe();
    const p = MOCK_PROJECTS.find((x) => x.id === projectId);
    if (!p) throw new Error('Project not found');
    if (!me || p.owner_id !== me.id) throw new Error('Only project owner can delete updates');
    const list = getProjectUpdatesList(projectId);
    const idx = list.findIndex((u) => u.id === updateId);
    if (idx < 0) throw new Error('Update not found');
    list.splice(idx, 1);
  },
};
