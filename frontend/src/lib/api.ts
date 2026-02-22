const API_URL = import.meta.env.PUBLIC_API_URL || "http://localhost:8080";
const MOCK_MODE = import.meta.env.PUBLIC_MOCK_MODE === "true";

/** プラットフォームプロジェクト（GIVErS への寄付）の ID。モック時は mock-4 */
export const PLATFORM_PROJECT_ID = "mock-4";

export async function fetchApi<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });

  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`);
  }

  return res.json();
}

export async function healthCheck(): Promise<{
  status: string;
  message: string;
}> {
  if (MOCK_MODE) return (await import("./mock-api")).mockApi.healthCheck();
  return fetchApi("/api/health");
}

export interface User {
  id: string;
  email: string;
  name: string;
  created_at: string;
  updated_at: string;
  /** ロール（モック時のみ。host=ホスト, project_owner=プロジェクトオーナー, donor=寄付者のみ） */
  role?: "host" | "project_owner" | "donor";
  /** トークンで記録された寄付をアカウントに引き継ぐ待ちがあれば true（getMe で返却。migrate-from-token で解消） */
  pending_token_migration?: boolean;
  /** ホストにより利用停止されている場合 true（getMe で返却。寄付・作成等は不可） */
  suspended?: boolean;
}

/** モック時のログイン切り替え用 localStorage キー */
export const MOCK_LOGIN_MODE_KEY = "givers_mock_login_mode";

/** 管理画面用ユーザー（status 付き） */
export interface AdminUser extends User {
  status: "active" | "suspended";
  project_count?: number;
}

/** ユーザー一覧（ホスト用、モック時のみ） */
export async function getAdminUsers(): Promise<AdminUser[]> {
  if (MOCK_MODE) return (await import("./mock-api")).mockApi.getAdminUsers();
  return [];
}

/** 開示用データ出力（ホストのみ）。第三者情報開示請求等に備えたエクスポート。 */
export interface DisclosureExportPayload {
  exported_at: string;
  platform: string;
  type: "user" | "project";
  user?: {
    id: string;
    email: string;
    name: string;
    created_at: string;
    updated_at: string;
    status: string;
    role?: string;
  };
  user_projects?: {
    id: string;
    name: string;
    status: string;
    created_at: string;
  }[];
  user_donations?: {
    id: string;
    project_id: string;
    project_name: string;
    amount: number;
    created_at: string;
  }[];
  user_recurring?: {
    id: string;
    project_id: string;
    project_name: string;
    amount: number;
    created_at: string;
    status: string;
    interval?: string;
  }[];
  project?: {
    id: string;
    name: string;
    description: string;
    owner_id: string;
    status: string;
    created_at: string;
    owner_name?: string;
  };
  project_donations?: {
    id: string;
    amount: number;
    created_at: string;
    donor_type?: string;
  }[];
}

export async function getDisclosureExport(
  type: "user" | "project",
  id: string,
): Promise<DisclosureExportPayload> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.getDisclosureExport(type, id);
  const res = await fetch(
    `${API_URL}/api/admin/disclosure-export?type=${type}&id=${encodeURIComponent(id)}`,
    {
      credentials: "include",
    },
  );
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return res.json();
}

export async function getMe(): Promise<User | null> {
  if (MOCK_MODE) return (await import("./mock-api")).mockApi.getMe();
  const res = await fetch(`${API_URL}/api/me`, {
    credentials: "include",
  });
  if (res.status === 401) return null;
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return res.json();
}

/** トークンで記録された寄付をアカウントに引き継ぐ。冪等。 */
export async function migrateFromToken(): Promise<{
  migrated_count: number;
  already_migrated?: boolean;
}> {
  if (MOCK_MODE) return (await import("./mock-api")).mockApi.migrateFromToken();
  return fetchApi("/api/me/migrate-from-token", {
    method: "POST",
    body: JSON.stringify({}),
  });
}

export interface AuthProviders {
  providers: string[];
}

export async function getAuthProviders(): Promise<AuthProviders> {
  if (MOCK_MODE) return (await import("./mock-api")).mockApi.getAuthProviders();
  return fetchApi("/api/auth/providers");
}

export async function getGoogleLoginUrl(): Promise<{ url: string }> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.getGoogleLoginUrl();
  return fetchApi("/api/auth/google/login");
}

export async function getGitHubLoginUrl(): Promise<{ url: string }> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.getGitHubLoginUrl();
  return fetchApi("/api/auth/github/login");
}

export async function getAppleLoginUrl(): Promise<{ url: string }> {
  if (MOCK_MODE) return (await import("./mock-api")).mockApi.getAppleLoginUrl();
  return fetchApi("/api/auth/apple/login");
}

export async function getEmailLoginUrl(): Promise<{ url: string }> {
  if (MOCK_MODE) return (await import("./mock-api")).mockApi.getEmailLoginUrl();
  return fetchApi("/api/auth/email/login");
}

export async function logout(): Promise<void> {
  if (MOCK_MODE) return (await import("./mock-api")).mockApi.logout();
  await fetch(`${API_URL}/api/auth/logout`, {
    method: "POST",
    credentials: "include",
  });
}

// --- Contact API ---

export interface ContactInput {
  email: string;
  name?: string;
  message: string;
}

export async function submitContact(
  input: ContactInput,
): Promise<{ ok: boolean }> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.submitContact(input);
  return fetchApi("/api/contact", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export interface ContactMessage {
  id: string;
  email: string;
  name?: string | null;
  message: string;
  status: "unread" | "read";
  created_at: string;
  updated_at: string;
}

export async function getAdminContacts(params?: {
  status?: string;
  limit?: number;
  offset?: number;
}): Promise<ContactMessage[]> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.getAdminContacts(params);
  const q = new URLSearchParams();
  if (params?.status) q.set("status", params.status);
  if (params?.limit != null) q.set("limit", String(params.limit));
  if (params?.offset != null) q.set("offset", String(params.offset));
  const qs = q.toString();
  const res = await fetchApi<{ messages: ContactMessage[] }>(
    `/api/admin/contacts${qs ? "?" + qs : ""}`,
  );
  return res.messages ?? [];
}

/** 問い合わせメッセージを既読にする（ホスト用） */
export async function markContactRead(id: string): Promise<void> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.markContactRead(id);
  await fetchApi(`/api/admin/contacts/${id}/status`, {
    method: "PATCH",
    body: JSON.stringify({ status: "read" }),
  });
}

// --- Legal API ---

export interface LegalDoc {
  type: string;
  content: string;
}

export async function getLegalDoc(
  type: "terms" | "privacy" | "disclaimer",
): Promise<LegalDoc | null> {
  if (MOCK_MODE) return (await import("./mock-api")).mockApi.getLegalDoc(type);
  try {
    return await fetchApi<LegalDoc>(`/api/legal/${type}`);
  } catch {
    return null;
  }
}

// --- Projects API ---

export interface ProjectCosts {
  id?: string;
  project_id?: string;
  server_cost_monthly: number;
  dev_cost_per_day: number;
  dev_days_per_month: number;
  other_cost_monthly: number;
}

export interface ProjectAlerts {
  id?: string;
  project_id?: string;
  warning_threshold: number;
  critical_threshold: number;
}

export interface Project {
  id: string;
  owner_id: string;
  name: string;
  description: string;
  deadline?: string | null;
  status: string;
  owner_want_monthly?: number | null;
  created_at: string;
  updated_at: string;
  costs?: ProjectCosts | null;
  alerts?: ProjectAlerts | null;
  /** 月額寄付の現在合計（モック/Phase4以降） */
  current_monthly_donations?: number;
  /** オーナー表示名（モック/Phase4以降） */
  owner_name?: string;
  /** 最近の応援者（モック/Phase4以降、匿名は null） */
  recent_supporters?: { name: string | null; amount: number }[];
  /** ヒーロー画像URL（モック/Phase5以降） */
  image_url?: string | null;
  /** プロジェクト概要・詳細説明（2000文字程度、モック/Phase5以降） */
  overview?: string | null;
  /** Stripe Connect URL（プロジェクト作成直後のみ、一時的） */
  stripe_connect_url?: string;
  /** 月額目標（cost_items の合計） */
  monthly_target?: number;
}

/** プロジェクトオーナーからのアップデート（モック/Phase5以降） */
export interface ProjectUpdate {
  id: string;
  project_id: string;
  created_at: string;
  title?: string | null;
  body: string;
  author_name?: string | null;
  /** 非表示フラグ。false のとき他ユーザーには見えず、オーナーにはグレーアウト＋再表示で表示 */
  visible?: boolean;
}

/** 金額表示タイプ: 希望額 / 必要額 / 両方 */
export type AmountInputType = "want" | "cost" | "both";

export async function getProjects(limit = 20, offset = 0): Promise<Project[]> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.getProjects(limit, offset);
  return fetchApi<Project[]>(`/api/projects?limit=${limit}&offset=${offset}`);
}

export async function getProject(id: string): Promise<Project> {
  if (MOCK_MODE) return (await import("./mock-api")).mockApi.getProject(id);
  return fetchApi<Project>(`/api/projects/${id}`);
}

export async function getMyProjects(): Promise<Project[]> {
  if (MOCK_MODE) return (await import("./mock-api")).mockApi.getMyProjects();
  return fetchApi<Project[]>("/api/me/projects");
}

// --- Checkout (Stripe 決済) ---

export interface CheckoutRequest {
  project_id: string;
  amount: number;
  currency?: string;
  is_recurring: boolean;
  message?: string;
  locale?: string;
  donor_token?: string;
}

export async function createCheckout(
  req: CheckoutRequest,
): Promise<{ checkout_url: string }> {
  if (MOCK_MODE) {
    // モック: フェイク URL を返す
    return { checkout_url: `/projects/${req.project_id}?donated=1` };
  }
  return fetchApi<{ checkout_url: string }>("/api/donations/checkout", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

// --- Donations (マイページ用、モック時のみ) ---

/** 寄付履歴 */
export interface Donation {
  id: string;
  user_id: string;
  project_id: string;
  project_name: string;
  amount: number;
  created_at: string;
  message?: string | null;
}

/** 定期寄付 */
export interface RecurringDonation {
  id: string;
  user_id: string;
  project_id: string;
  project_name: string;
  amount: number;
  created_at: string;
  status: "active" | "paused" | "cancelled";
  /** 寄付タイミング（月額/年額など） */
  interval?: "monthly" | "yearly";
}

export async function getMyDonations(): Promise<Donation[]> {
  if (MOCK_MODE) return (await import("./mock-api")).mockApi.getMyDonations();
  return [];
}

export async function getMyRecurringDonations(): Promise<RecurringDonation[]> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.getMyRecurringDonations();
  return [];
}

export async function cancelRecurringDonation(id: string): Promise<void> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.cancelRecurringDonation(id);
}

/** 定期寄付の変更（金額・タイミング） */
export async function updateRecurringDonation(
  id: string,
  input: { amount?: number; interval?: "monthly" | "yearly" },
): Promise<RecurringDonation> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.updateRecurringDonation(
      id,
      input,
    );
  throw new Error("Not implemented");
}

/** 定期寄付の一時休止 */
export async function pauseRecurringDonation(id: string): Promise<void> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.pauseRecurringDonation(id);
  throw new Error("Not implemented");
}

/** 定期寄付の再開 */
export async function resumeRecurringDonation(id: string): Promise<void> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.resumeRecurringDonation(id);
  throw new Error("Not implemented");
}

/** 定期寄付の削除（完全に解除） */
export async function deleteRecurringDonation(id: string): Promise<void> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.deleteRecurringDonation(id);
  throw new Error("Not implemented");
}

/** 新着プロジェクト（モック時のみ、トップページ用） */
export async function getNewProjects(limit = 5): Promise<Project[]> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.getNewProjects(limit);
  return getProjects(limit, 0);
}

/** HOT プロジェクト（モック時のみ、トップページ用） */
export async function getHotProjects(limit = 5): Promise<Project[]> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.getHotProjects(limit);
  return getProjects(limit, 0);
}

/** 関連プロジェクト（プロジェクト詳細用。当該を除く HOT 等） */
export async function getRelatedProjects(
  projectId: string,
  limit = 4,
): Promise<Project[]> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.getRelatedProjects(
      projectId,
      limit,
    );
  const all = await getProjects(limit + 5, 0);
  return all.filter((p) => p.id !== projectId).slice(0, limit);
}

/** ウォッチ一覧（ログインユーザーがウォッチしているプロジェクト） */
export async function getWatchedProjects(): Promise<Project[]> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.getWatchedProjects();
  const res = await fetch(`${API_URL}/api/me/watches`, {
    credentials: "include",
  });
  if (res.status === 401) return [];
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return res.json();
}

/** プロジェクトをウォッチする */
export async function watchProject(projectId: string): Promise<void> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.watchProject(projectId);
  await fetchApi(`/api/projects/${projectId}/watch`, { method: "POST" });
}

/** プロジェクトのウォッチを解除する */
export async function unwatchProject(projectId: string): Promise<void> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.unwatchProject(projectId);
  await fetchApi(`/api/projects/${projectId}/watch`, { method: "DELETE" });
}

// --- Activity Feed (モック時のみ) ---

export type ActivityType =
  | "project_created"
  | "project_updated"
  | "donation"
  | "milestone";

export interface ActivityItem {
  id: string;
  type: ActivityType;
  created_at: string;
  project_id: string;
  project_name: string;
  actor_name: string | null;
  amount?: number;
  rate?: number;
}

export async function getActivityFeed(limit = 10): Promise<ActivityItem[]> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.getActivityFeed(limit);
  return [];
}

// --- Project Chart (モック時のみ) ---

export interface ChartDataPoint {
  month: string;
  minAmount: number;
  targetAmount: number;
  actualAmount: number;
}

export async function getProjectChart(
  projectId: string,
): Promise<ChartDataPoint[]> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.getProjectChart(projectId);
  return [];
}

/** プロジェクトオーナーからのアップデート（モック時のみ） */
export async function getProjectUpdates(
  projectId: string,
  limit = 20,
): Promise<ProjectUpdate[]> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.getProjectUpdates(
      projectId,
      limit,
    );
  return [];
}

/** アップデート投稿（モック時のみ。オーナー限定） */
export interface CreateProjectUpdateInput {
  title?: string | null;
  body: string;
}

export async function createProjectUpdate(
  projectId: string,
  input: CreateProjectUpdateInput,
): Promise<ProjectUpdate> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.createProjectUpdate(
      projectId,
      input,
    );
  throw new Error("Not implemented");
}

/** アップデート編集（モック時のみ。オーナー限定。visible で非表示/再表示） */
export async function updateProjectUpdate(
  projectId: string,
  updateId: string,
  input: { title?: string | null; body?: string; visible?: boolean },
): Promise<ProjectUpdate> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.updateProjectUpdate(
      projectId,
      updateId,
      input,
    );
  throw new Error("Not implemented");
}

/** アップデート非表示（モック時のみ。オーナー限定。実態は visible: false に更新） */
export async function deleteProjectUpdate(
  projectId: string,
  updateId: string,
): Promise<void> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.deleteProjectUpdate(
      projectId,
      updateId,
    );
  throw new Error("Not implemented");
}

export interface CreateProjectInput {
  name: string;
  description?: string;
  deadline?: string | null;
  status?: string;
  owner_want_monthly?: number | null;
  costs?: ProjectCosts | null;
  alerts?: ProjectAlerts | null;
}

export async function createProject(
  input: CreateProjectInput,
): Promise<Project> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.createProject(input);
  return fetchApi<Project>("/api/projects", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export interface UpdateProjectInput {
  name?: string;
  description?: string;
  overview?: string | null;
  deadline?: string | null;
  status?: string;
  owner_want_monthly?: number | null;
  costs?: ProjectCosts | null;
  alerts?: ProjectAlerts | null;
}

export async function updateProject(
  id: string,
  input: UpdateProjectInput,
): Promise<Project> {
  if (MOCK_MODE)
    return (await import("./mock-api")).mockApi.updateProject(id, input);
  return fetchApi<Project>(`/api/projects/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}
