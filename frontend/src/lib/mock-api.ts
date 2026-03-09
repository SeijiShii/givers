import type {
  Project,
  ProjectAlerts,
  CreateProjectInput,
  UpdateProjectInput,
  User,
  Donation,
  RecurringDonation,
  AuthProviders,
  ContactInput,
  ContactMessage,
  LegalDoc,
  MessageThread,
  Message,
  MessageThreadListResult,
  MessageListResult,
} from "./api";
import type { AdminUser, DisclosureExportPayload } from "./api";
import { MOCK_LOGIN_MODE_KEY } from "./api";

/** モック: トークン→アカウント移行済みフラグ（localStorage）。true なら getMe で pending_token_migration を返さない */
const MOCK_MIGRATION_DONE_KEY = "givers_mock_migration_done";
/** モック: 利用停止ユーザーをシミュレート（localStorage）。true なら getMe で suspended: true を返す */
const MOCK_SUSPENDED_USER_KEY = "givers_mock_suspended_user";
import {
  MOCK_HOST_USER,
  MOCK_MEMBER_USER,
  MOCK_DONOR_USER,
  MOCK_ADMIN_USERS,
} from "../data/mock-users";
import type { ProjectUpdate, CreateProjectUpdateInput } from "./api";
import { MOCK_PROJECTS, type MockProject } from "../data/mock-projects";
import {
  MOCK_ACTIVITIES,
  MOCK_OWNERS,
  MOCK_RECENT_SUPPORTERS,
  type ActivityItem,
} from "../data/mock-activities";
import { MOCK_CHART_DATA, type ChartDataPoint } from "../data/mock-chart-data";
import { MOCK_PROJECT_UPDATES } from "../data/mock-project-updates";
import {
  MOCK_DONATIONS,
  MOCK_RECURRING_DONATIONS,
} from "../data/mock-donations";

const delay = (ms: number) => new Promise((r) => setTimeout(r, ms));

/** モック用の擬似遅延（体感用、0 にしても可） */
const MOCK_DELAY = 150;

/** 問い合わせ既読 ID（セッション内のみ、モック用） */
const readContactIds = new Set<string>(["contact-mock-2"]);

/** 定期寄付キャンセル済み ID（セッション内のみ、モック用） */
const cancelledRecurringIds = new Set<string>();

/** 定期寄付一時休止 ID（セッション内のみ、モック用） */
const pausedRecurringIds = new Set<string>();

/** 定期寄付削除済み ID（セッション内のみ、モック用。一覧から非表示） */
const deletedRecurringIds = new Set<string>();

/** 定期寄付の変更（金額・タイミング）上書き（セッション内のみ、モック用） */
const recurringOverrides = new Map<
  string,
  { amount?: number; interval?: "monthly" | "yearly" }
>();

/** モック: お知らせ既読 ID（セッション内のみ） */
const mockAnnouncementReadIds = new Set<string>();

/** モック: お知らせデータ */
const MOCK_ANNOUNCEMENTS: Array<{
  id: string;
  author_id: string;
  title: string;
  body: string;
  severity: "info" | "warn" | "error";
  visible: boolean;
  published_at: string;
  created_at: string;
  updated_at: string;
}> = [
  {
    id: "ann-1",
    author_id: "mock-host",
    title: "サーバーメンテナンスのお知らせ",
    body: "2026年3月10日 02:00〜04:00 にサーバーメンテナンスを実施します。この間、サービスが一時的にご利用いただけなくなります。",
    severity: "warn",
    visible: true,
    published_at: "2026-03-03T10:00:00Z",
    created_at: "2026-03-03T10:00:00Z",
    updated_at: "2026-03-03T10:00:00Z",
  },
  {
    id: "ann-2",
    author_id: "mock-host",
    title: "新機能リリースのお知らせ",
    body: "返金機能を追加しました。寄付者・プロジェクトオーナーともに寄付の返金が可能になりました。",
    severity: "info",
    visible: true,
    published_at: "2026-02-28T09:00:00Z",
    created_at: "2026-02-28T09:00:00Z",
    updated_at: "2026-02-28T09:00:00Z",
  },
  {
    id: "ann-3",
    author_id: "mock-host",
    title: "決済遅延のお知らせ",
    body: "現在Stripe側の遅延により、一部の決済処理に時間がかかっています。復旧次第お知らせします。",
    severity: "error",
    visible: true,
    published_at: "2026-02-25T14:00:00Z",
    created_at: "2026-02-25T14:00:00Z",
    updated_at: "2026-02-25T14:00:00Z",
  },
];

/** プロジェクトステータス上書き（セッション内のみ、モック用） */
const projectStatusOverrides = new Map<string, string>();

/** プロジェクト概要上書き（セッション内のみ、モック用） */
const projectOverviewOverrides = new Map<string, string>();

/** プロジェクトアップデート（初期値 + セッション内投稿、モック用） */
const projectUpdatesStore = new Map<string, ProjectUpdate[]>();

/** モック: ウォッチ一覧（localStorage）。{ [userId]: projectId[] } */
const MOCK_WATCHED_KEY = "givers_watched_projects";

function getWatchedIds(userId: string): string[] {
  if (typeof window === "undefined" || !window.localStorage) return [];
  try {
    const raw = window.localStorage.getItem(MOCK_WATCHED_KEY);
    if (!raw) return [];
    const obj = JSON.parse(raw) as Record<string, string[]>;
    return obj[userId] ?? [];
  } catch {
    return [];
  }
}

function setWatchedIds(userId: string, ids: string[]): void {
  if (typeof window === "undefined" || !window.localStorage) return;
  try {
    const raw = window.localStorage.getItem(MOCK_WATCHED_KEY);
    const obj = (raw ? JSON.parse(raw) : {}) as Record<string, string[]>;
    obj[userId] = ids;
    window.localStorage.setItem(MOCK_WATCHED_KEY, JSON.stringify(obj));
  } catch {
    // ignore
  }
}

function getProjectUpdatesList(projectId: string): ProjectUpdate[] {
  if (!projectUpdatesStore.has(projectId)) {
    const initial = (MOCK_PROJECT_UPDATES[projectId] ?? []).map((u) => ({
      ...u,
      visible: u.visible ?? true,
    }));
    projectUpdatesStore.set(projectId, initial);
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
  async getMe(): Promise<User | null> {
    await delay(MOCK_DELAY);
    if (typeof window === "undefined" || !window.localStorage) {
      return MOCK_HOST_USER;
    }
    const mode = window.localStorage.getItem(MOCK_LOGIN_MODE_KEY);
    if (mode === "logout") return null;
    if (mode === "donor") {
      const migrationDone =
        window.localStorage.getItem(MOCK_MIGRATION_DONE_KEY) === "true";
      const suspended =
        window.localStorage.getItem(MOCK_SUSPENDED_USER_KEY) === "true";
      return {
        ...MOCK_DONOR_USER,
        pending_token_migration: migrationDone ? undefined : true,
        ...(suspended ? { suspended: true as const } : {}),
      };
    }
    if (mode === "project_owner" || mode === "member") {
      const suspended =
        window.localStorage.getItem(MOCK_SUSPENDED_USER_KEY) === "true";
      return {
        ...MOCK_MEMBER_USER,
        ...(suspended ? { suspended: true as const } : {}),
      };
    }
    const suspended =
      window.localStorage.getItem(MOCK_SUSPENDED_USER_KEY) === "true";
    return {
      ...MOCK_HOST_USER,
      ...(suspended ? { suspended: true as const } : {}),
    };
  },

  async migrateFromToken(): Promise<{
    migrated_count: number;
    already_migrated?: boolean;
  }> {
    await delay(MOCK_DELAY);
    if (typeof window !== "undefined" && window.localStorage) {
      if (window.localStorage.getItem(MOCK_MIGRATION_DONE_KEY) === "true") {
        return { migrated_count: 0, already_migrated: true };
      }
      window.localStorage.setItem(MOCK_MIGRATION_DONE_KEY, "true");
    }
    return { migrated_count: 1, already_migrated: false };
  },

  async getGoogleLoginUrl(): Promise<{ url: string }> {
    await delay(MOCK_DELAY);
    return { url: "#" };
  },

  async getGitHubLoginUrl(): Promise<{ url: string }> {
    await delay(MOCK_DELAY);
    return { url: "#" };
  },

  async getAppleLoginUrl(): Promise<{ url: string }> {
    await delay(MOCK_DELAY);
    return { url: "#" };
  },

  async getDiscordLoginUrl(): Promise<{ url: string }> {
    await delay(MOCK_DELAY);
    return { url: "#" };
  },

  async getEmailLoginUrl(): Promise<{ url: string }> {
    await delay(MOCK_DELAY);
    return { url: "#" };
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
    if (!p) throw new Error("Project not found");
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
    return [...list].sort(
      (a, b) =>
        new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
    );
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
        if (cancelledRecurringIds.has(r.id)) status = "cancelled";
        else if (pausedRecurringIds.has(r.id)) status = "paused";
        return {
          ...r,
          ...overrides,
          status,
          interval: overrides?.interval ?? r.interval ?? "monthly",
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
    input: { amount?: number; interval?: "monthly" | "yearly" },
  ): Promise<RecurringDonation> {
    await delay(MOCK_DELAY);
    const me = await this.getMe();
    if (!me) throw new Error("Not logged in");
    const list = (MOCK_RECURRING_DONATIONS[me.id] ?? []).filter(
      (r) => !deletedRecurringIds.has(r.id),
    );
    const r = list.find((x) => x.id === id);
    if (!r) throw new Error("Recurring donation not found");
    const current = recurringOverrides.get(id) ?? {};
    recurringOverrides.set(id, { ...current, ...input });
    const all = await this.getMyRecurringDonations();
    const updated = all.find((u) => u.id === id);
    if (!updated) throw new Error("Recurring donation not found");
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
    const ownerId = me?.id ?? "user-mock";
    const id = `mock-new-${Date.now()}`;
    const now = new Date().toISOString();
    // cost_items から monthly_target を計算
    const items = input.cost_items ?? [];
    const mt = items.reduce((sum, ci) => sum + ci.unit_price * ci.quantity, 0);
    const newProject: Project = {
      id,
      owner_id: ownerId,
      name: input.name,
      description: input.description ?? "",
      overview: input.overview ?? "",
      status: input.status ?? "active",
      owner_want_monthly: input.owner_want_monthly ?? null,
      created_at: now,
      updated_at: now,
      cost_items: items.length > 0 ? items : null,
      monthly_target: mt,
      alerts: input.alerts ?? null,
    };
    return newProject;
  },

  async updateProject(id: string, input: UpdateProjectInput): Promise<Project> {
    await delay(MOCK_DELAY);
    const p = MOCK_PROJECTS.find((x) => x.id === id);
    if (!p) throw new Error("Project not found");
    if (input.status != null) projectStatusOverrides.set(id, input.status);
    if (input.overview != null)
      projectOverviewOverrides.set(id, input.overview);
    const merged = {
      ...p,
      ...input,
      id: p.id,
      owner_id: p.owner_id,
      created_at: p.created_at,
      status: input.status ?? projectStatusOverrides.get(id) ?? p.status,
    };
    const project = toProject(merged as typeof p);
    const overviewOverride = projectOverviewOverrides.get(id);
    if (overviewOverride != null) project.overview = overviewOverride;
    return project;
  },

  /** 新着プロジェクト（created_at 降順） */
  async getNewProjects(limit = 5): Promise<Project[]> {
    await delay(MOCK_DELAY);
    const sorted = [...MOCK_PROJECTS].sort(
      (a, b) =>
        new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
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

  /** ウォッチ一覧（ログインユーザーがウォッチしているプロジェクト） */
  async getWatchedProjects(): Promise<Project[]> {
    await delay(MOCK_DELAY);
    const me = await this.getMe();
    if (!me) return [];
    const ids = getWatchedIds(me.id);
    const projects: Project[] = [];
    for (const id of ids) {
      const p = MOCK_PROJECTS.find((x) => x.id === id);
      if (p) projects.push(toProject(p));
    }
    return projects;
  },

  /** プロジェクトをウォッチする */
  async watchProject(projectId: string): Promise<void> {
    await delay(MOCK_DELAY);
    const me = await this.getMe();
    if (!me) return;
    const ids = getWatchedIds(me.id);
    if (ids.includes(projectId)) return;
    setWatchedIds(me.id, [...ids, projectId]);
  },

  /** プロジェクトのウォッチを解除する */
  async unwatchProject(projectId: string): Promise<void> {
    await delay(MOCK_DELAY);
    const me = await this.getMe();
    if (!me) return;
    const ids = getWatchedIds(me.id).filter((id) => id !== projectId);
    setWatchedIds(me.id, ids);
  },

  /** アクティビティフィード */
  async getActivityFeed(limit = 10): Promise<ActivityItem[]> {
    await delay(MOCK_DELAY);
    const sorted = [...MOCK_ACTIVITIES].sort(
      (a, b) =>
        new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
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

  async searchUsers(q: string): Promise<AdminUser[]> {
    await delay(MOCK_DELAY);
    const lower = q.toLowerCase();
    return MOCK_ADMIN_USERS.filter(
      (u) =>
        u.name.toLowerCase().includes(lower) ||
        u.email.toLowerCase().includes(lower),
    );
  },

  async getAdminUser(id: string): Promise<AdminUser | null> {
    await delay(MOCK_DELAY);
    return MOCK_ADMIN_USERS.find((u) => u.id === id) ?? null;
  },

  async suspendUser(id: string, suspended: boolean): Promise<{ ok: boolean }> {
    await delay(MOCK_DELAY);
    const user = MOCK_ADMIN_USERS.find((u) => u.id === id);
    if (user) {
      (user as AdminUser).status = suspended ? "suspended" : "active";
    }
    return { ok: true };
  },

  /** 開示用データ出力（ホスト用。第三者情報開示請求等に備える） */
  async getDisclosureExport(
    type: "user" | "project",
    id: string,
  ): Promise<DisclosureExportPayload> {
    await delay(MOCK_DELAY);
    const exported_at = new Date().toISOString();
    const platform = "GIVErS";

    if (type === "user") {
      const user = MOCK_ADMIN_USERS.find((u) => u.id === id);
      if (!user) throw new Error("User not found");
      const user_projects = MOCK_PROJECTS.filter((p) => p.owner_id === id).map(
        (p) => ({
          id: p.id,
          name: p.name,
          status: p.status,
          created_at: p.created_at,
        }),
      );
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
        type: "user",
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
    if (!p) throw new Error("Project not found");
    const owner = MOCK_ADMIN_USERS.find((u) => u.id === p.owner_id);
    const allDonations = Object.values(MOCK_DONATIONS).flat();
    const projectDonations = allDonations
      .filter((d) => d.project_id === id)
      .map((d) => ({
        id: d.id,
        amount: d.amount,
        created_at: d.created_at,
        donor_type: "user" as const,
      }));
    return {
      exported_at,
      platform,
      type: "project",
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
  async getProjectUpdates(
    projectId: string,
    limit = 20,
  ): Promise<ProjectUpdate[]> {
    await delay(MOCK_DELAY);
    const list = getProjectUpdatesList(projectId);
    return list
      .sort(
        (a, b) =>
          new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
      )
      .slice(0, limit);
  },

  /** アップデート投稿（オーナー限定） */
  async createProjectUpdate(
    projectId: string,
    input: CreateProjectUpdateInput,
  ): Promise<ProjectUpdate> {
    await delay(MOCK_DELAY);
    const me = await this.getMe();
    const p = MOCK_PROJECTS.find((x) => x.id === projectId);
    if (!p) throw new Error("Project not found");
    if (!me || p.owner_id !== me.id)
      throw new Error("Only project owner can post updates");
    const list = getProjectUpdatesList(projectId);
    const now = new Date().toISOString();
    const newUpdate: ProjectUpdate = {
      id: `up-${projectId}-${Date.now()}`,
      project_id: projectId,
      created_at: now,
      title: input.title ?? null,
      body: input.body,
      author_name: me.name,
      visible: true,
    };
    list.unshift(newUpdate);
    return newUpdate;
  },

  /** アップデート編集（オーナー限定。visible で非表示/再表示） */
  async updateProjectUpdate(
    projectId: string,
    updateId: string,
    input: { title?: string | null; body?: string; visible?: boolean },
  ): Promise<ProjectUpdate> {
    await delay(MOCK_DELAY);
    const me = await this.getMe();
    const p = MOCK_PROJECTS.find((x) => x.id === projectId);
    if (!p) throw new Error("Project not found");
    if (!me || p.owner_id !== me.id)
      throw new Error("Only project owner can edit updates");
    const list = getProjectUpdatesList(projectId);
    const idx = list.findIndex((u) => u.id === updateId);
    if (idx < 0) throw new Error("Update not found");
    const updated = { ...list[idx], ...input };
    list[idx] = updated;
    return updated;
  },

  /** アップデート非表示（オーナー限定。visible: false に更新し、他ユーザーには見えなくする） */
  async deleteProjectUpdate(
    projectId: string,
    updateId: string,
  ): Promise<void> {
    await delay(MOCK_DELAY);
    const me = await this.getMe();
    const p = MOCK_PROJECTS.find((x) => x.id === projectId);
    if (!p) throw new Error("Project not found");
    if (!me || p.owner_id !== me.id)
      throw new Error("Only project owner can hide updates");
    const list = getProjectUpdatesList(projectId);
    const idx = list.findIndex((u) => u.id === updateId);
    if (idx < 0) throw new Error("Update not found");
    list[idx] = { ...list[idx], visible: false };
  },

  /** 認証プロバイダー一覧（モックでは google + github を返す） */
  async getAuthProviders(): Promise<AuthProviders> {
    await delay(MOCK_DELAY);
    return { providers: ["google", "github", "discord"] };
  },

  /** お問い合わせ送信（モックでは常に成功） */
  async submitContact(_input: ContactInput): Promise<{ ok: boolean }> {
    await delay(MOCK_DELAY);
    return { ok: true };
  },

  /** お問い合わせ一覧（ホスト用） */
  async getAdminContacts(params?: {
    status?: string;
    limit?: number;
    offset?: number;
  }): Promise<ContactMessage[]> {
    await delay(MOCK_DELAY);
    const base: ContactMessage[] = [
      {
        id: "contact-mock-1",
        email: "sender@example.com",
        name: "テストユーザー",
        message: "これはモックのお問い合わせメッセージです。",
        status: readContactIds.has("contact-mock-1") ? "read" : "unread",
        created_at: "2026-02-21T10:00:00Z",
        updated_at: "2026-02-21T10:00:00Z",
      },
      {
        id: "contact-mock-2",
        email: "another@example.com",
        name: null,
        message: "匿名のお問い合わせです。返信不要で結構です。",
        status: readContactIds.has("contact-mock-2") ? "read" : "unread",
        created_at: "2026-02-20T15:30:00Z",
        updated_at: "2026-02-20T15:30:00Z",
      },
    ];
    const filtered =
      params?.status === "unread"
        ? base.filter((m) => m.status === "unread")
        : params?.status === "read"
          ? base.filter((m) => m.status === "read")
          : base;
    const offset = params?.offset ?? 0;
    const limit = params?.limit ?? 20;
    return filtered.slice(offset, offset + limit);
  },

  /** 問い合わせを既読にする（モック: セッション内 Set で管理） */
  async markContactRead(id: string): Promise<void> {
    await delay(MOCK_DELAY);
    readContactIds.add(id);
  },

  /** 問い合わせを未読に戻す（モック） */
  async markContactUnread(id: string): Promise<void> {
    await delay(MOCK_DELAY);
    readContactIds.delete(id);
  },

  /** 法的文書取得（モックでは固定テキストを返す） */
  async getLegalDoc(
    type: "terms" | "privacy" | "disclaimer" | "commerce-law",
    locale?: string,
  ): Promise<LegalDoc | null> {
    await delay(MOCK_DELAY);
    const lang = locale === "ja" ? "ja" : "en";
    const labelsJa: Record<string, string> = {
      terms: "利用規約",
      privacy: "プライバシーポリシー",
      disclaimer: "免責事項",
      "commerce-law": "特定商取引法に基づく表記",
    };
    const labelsEn: Record<string, string> = {
      terms: "Terms of Service",
      privacy: "Privacy Policy",
      disclaimer: "Disclaimer",
      "commerce-law": "Specified Commercial Transactions Act Disclosure",
    };
    const labels = lang === "ja" ? labelsJa : labelsEn;
    const suffix =
      lang === "ja"
        ? "これはモックのコンテンツです。\n\n実際のコンテンツはサーバー上の Markdown ファイルから読み込まれます。"
        : "This is mock content.\n\nActual content is loaded from Markdown files on the server.";
    return {
      type,
      content: `# ${labels[type] ?? type}\n\n${suffix}`,
    };
  },

  async uploadProjectImage(
    projectId: string,
    _file: File,
  ): Promise<{ image_url: string }> {
    await delay(MOCK_DELAY);
    const url = `https://placehold.co/800x400/4a7c59/ffffff?text=Uploaded`;
    const p = MOCK_PROJECTS.find((x) => x.id === projectId);
    if (p) (p as Record<string, unknown>)._mockImageUrl = url;
    return { image_url: url };
  },

  async deleteProjectImage(projectId: string): Promise<void> {
    await delay(MOCK_DELAY);
    const p = MOCK_PROJECTS.find((x) => x.id === projectId);
    if (p) (p as Record<string, unknown>)._mockImageUrl = null;
  },

  async refundDonationByOwner(
    _projectId: string,
    _donationId: string,
  ): Promise<{ status: string }> {
    await delay(MOCK_DELAY);
    return { status: "refunded" };
  },

  async refundDonationByDonor(
    _donationId: string,
  ): Promise<{ status: string }> {
    await delay(MOCK_DELAY);
    return { status: "refunded" };
  },

  async getAnnouncements(limit: number, cursor?: string) {
    await delay(MOCK_DELAY);
    const all = MOCK_ANNOUNCEMENTS.filter(
      (a) => a.visible && new Date(a.published_at) <= new Date(),
    );
    return {
      announcements: all.slice(0, limit),
      next_cursor: null as string | null,
    };
  },

  async getAnnouncementUnreadCount() {
    await delay(MOCK_DELAY);
    return MOCK_ANNOUNCEMENTS.filter(
      (a) =>
        a.visible &&
        new Date(a.published_at) <= new Date() &&
        !mockAnnouncementReadIds.has(a.id),
    ).length;
  },

  async markAnnouncementRead(id: string) {
    await delay(MOCK_DELAY);
    mockAnnouncementReadIds.add(id);
  },

  async getAdminAnnouncements(limit: number, cursor?: string) {
    await delay(MOCK_DELAY);
    return {
      announcements: MOCK_ANNOUNCEMENTS.slice(0, limit),
      next_cursor: null as string | null,
    };
  },

  async createAnnouncement(input: {
    title: string;
    body?: string;
    severity?: string;
    published_at?: string;
  }) {
    await delay(MOCK_DELAY);
    const a = {
      id: "ann-" + Date.now(),
      author_id: "mock-host",
      title: input.title,
      body: input.body ?? "",
      severity: (input.severity ?? "info") as "info" | "warn" | "error",
      visible: true,
      published_at: input.published_at ?? new Date().toISOString(),
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    MOCK_ANNOUNCEMENTS.unshift(a);
    return a;
  },

  async updateAnnouncement(
    id: string,
    input: {
      title?: string;
      body?: string;
      severity?: string;
      published_at?: string;
    },
  ) {
    await delay(MOCK_DELAY);
    const a = MOCK_ANNOUNCEMENTS.find((x) => x.id === id);
    if (!a) throw new Error("not found");
    if (input.title !== undefined) a.title = input.title;
    if (input.body !== undefined) a.body = input.body;
    if (input.severity !== undefined)
      a.severity = input.severity as "info" | "warn" | "error";
    if (input.published_at !== undefined) a.published_at = input.published_at;
    a.updated_at = new Date().toISOString();
    return { ...a };
  },

  async setAnnouncementVisibility(id: string, visible: boolean) {
    await delay(MOCK_DELAY);
    const a = MOCK_ANNOUNCEMENTS.find((x) => x.id === id);
    if (a) a.visible = visible;
  },

  async getHostHealth(): Promise<{
    monthly_cost: number;
    current_monthly: number;
    warning_threshold: number;
    critical_threshold: number;
    rate: number;
    signal: "green" | "yellow" | "red";
  }> {
    await delay(MOCK_DELAY);
    const p = MOCK_PROJECTS.find((x) => x.id === "mock-4");
    const target = p?.owner_want_monthly ?? 0;
    const current = p?._mockCurrentMonthly ?? 0;
    const rate = target > 0 ? Math.round((current / target) * 100) : 0;
    const signal: "green" | "yellow" | "red" =
      rate >= 60 ? "green" : rate >= 30 ? "yellow" : "red";
    return {
      monthly_cost: target,
      current_monthly: current,
      warning_threshold: 60,
      critical_threshold: 30,
      rate,
      signal,
    };
  },

  /** Quick re-donate（モック: 常に succeeded を返す） */
  async quickDonate(
    _donationId: string,
  ): Promise<{ status: string; donation_id?: string }> {
    await delay(MOCK_DELAY);
    return { status: "succeeded", donation_id: `don-quick-${Date.now()}` };
  },

  // --- Messages ---

  async getMyThreads(
    limit = 20,
    _cursor?: string,
  ): Promise<MessageThreadListResult> {
    await delay(MOCK_DELAY);
    return { threads: MOCK_THREADS.slice(0, limit), next_cursor: null };
  },

  async getThread(id: string): Promise<MessageThread> {
    await delay(MOCK_DELAY);
    const t = MOCK_THREADS.find((x) => x.id === id);
    if (!t) throw new Error("not found");
    return { ...t };
  },

  async getThreadMessages(
    threadId: string,
    limit = 50,
    _cursor?: string,
  ): Promise<MessageListResult> {
    await delay(MOCK_DELAY);
    const msgs = MOCK_MESSAGES.filter((m) => m.thread_id === threadId).slice(
      0,
      limit,
    );
    return { messages: msgs, next_cursor: null };
  },

  async sendMessage(threadId: string, body: string): Promise<Message> {
    await delay(MOCK_DELAY);
    const msg: Message = {
      id: `msg-${Date.now()}`,
      thread_id: threadId,
      sender_id: "mock-user-1",
      sender_name: "テストユーザー",
      body,
      created_at: new Date().toISOString(),
    };
    MOCK_MESSAGES.push(msg);
    return msg;
  },

  async getMessageUnreadCount(): Promise<number> {
    await delay(MOCK_DELAY);
    return 2;
  },

  async broadcastThread(subject: string, body: string): Promise<number> {
    await delay(MOCK_DELAY);
    const users = MOCK_ADMIN_USERS.filter((u) => u.id !== "mock-host-1");
    for (const u of users) {
      await this.createThread(u.id, subject, body);
    }
    return users.length;
  },

  async createThread(
    participantUserId: string,
    subject: string,
    body: string,
  ): Promise<MessageThread> {
    await delay(MOCK_DELAY);
    const thread: MessageThread = {
      id: `thread-${Date.now()}`,
      host_user_id: "mock-host-1",
      participant_user_id: participantUserId,
      subject,
      active: true,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      participant_name: "新規ユーザー",
      host_name: "ホスト",
      unread_count: 0,
      last_message_body: body,
      last_message_at: new Date().toISOString(),
    };
    MOCK_THREADS.unshift(thread);
    MOCK_MESSAGES.push({
      id: `msg-${Date.now()}`,
      thread_id: thread.id,
      sender_id: "mock-host-1",
      sender_name: "ホスト",
      body,
      created_at: new Date().toISOString(),
    });
    return thread;
  },

  async getAdminThreads(
    limit = 20,
    _cursor?: string,
  ): Promise<MessageThreadListResult> {
    await delay(MOCK_DELAY);
    return { threads: MOCK_THREADS.slice(0, limit), next_cursor: null };
  },

  async setThreadActive(threadId: string, active: boolean): Promise<void> {
    await delay(MOCK_DELAY);
    const t = MOCK_THREADS.find((x) => x.id === threadId);
    if (t) t.active = active;
  },
};

const MOCK_THREADS: MessageThread[] = [
  {
    id: "thread-1",
    host_user_id: "mock-host-1",
    participant_user_id: "mock-user-1",
    subject: "プロジェクトへのご支援について",
    active: true,
    created_at: "2026-03-01T10:00:00Z",
    updated_at: "2026-03-07T15:30:00Z",
    participant_name: "テストユーザー",
    host_name: "ホスト",
    unread_count: 1,
    last_message_body: "ご質問ありがとうございます。追加情報をお送りします。",
    last_message_at: "2026-03-07T15:30:00Z",
  },
  {
    id: "thread-2",
    host_user_id: "mock-host-1",
    participant_user_id: "mock-user-2",
    subject: "リワードについて",
    active: false,
    created_at: "2026-02-20T09:00:00Z",
    updated_at: "2026-02-25T12:00:00Z",
    participant_name: "ユーザー2",
    host_name: "ホスト",
    unread_count: 0,
    last_message_body: "承知しました。ありがとうございます。",
    last_message_at: "2026-02-25T12:00:00Z",
  },
];

const MOCK_MESSAGES: Message[] = [
  {
    id: "msg-1",
    thread_id: "thread-1",
    sender_id: "mock-host-1",
    sender_name: "ホスト",
    body: "こんにちは！プロジェクトへのご支援ありがとうございます。何かご質問がありましたらお気軽にどうぞ。",
    created_at: "2026-03-01T10:00:00Z",
  },
  {
    id: "msg-2",
    thread_id: "thread-1",
    sender_id: "mock-user-1",
    sender_name: "テストユーザー",
    body: "ありがとうございます！進捗について詳しく教えていただけますか？",
    created_at: "2026-03-05T14:00:00Z",
  },
  {
    id: "msg-3",
    thread_id: "thread-1",
    sender_id: "mock-host-1",
    sender_name: "ホスト",
    body: "ご質問ありがとうございます。追加情報をお送りします。",
    created_at: "2026-03-07T15:30:00Z",
  },
  {
    id: "msg-4",
    thread_id: "thread-2",
    sender_id: "mock-host-1",
    sender_name: "ホスト",
    body: "リワードの発送について確認させてください。",
    created_at: "2026-02-20T09:00:00Z",
  },
  {
    id: "msg-5",
    thread_id: "thread-2",
    sender_id: "mock-user-2",
    sender_name: "ユーザー2",
    body: "承知しました。ありがとうございます。",
    created_at: "2026-02-25T12:00:00Z",
  },
];
