const API_URL = import.meta.env.PUBLIC_API_URL || 'http://localhost:8080';
const MOCK_MODE = import.meta.env.PUBLIC_MOCK_MODE === 'true';

/** プラットフォームプロジェクト（GIVErS への寄付）の ID。モック時は mock-4 */
export const PLATFORM_PROJECT_ID = 'mock-4';

export async function fetchApi<T>(
  path: string,
  options?: RequestInit
): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`);
  }

  return res.json();
}

export async function healthCheck(): Promise<{ status: string; message: string }> {
  if (MOCK_MODE) return (await import('./mock-api')).mockApi.healthCheck();
  return fetchApi('/api/health');
}

export interface User {
  id: string;
  email: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export async function getMe(): Promise<User | null> {
  if (MOCK_MODE) return (await import('./mock-api')).mockApi.getMe();
  const res = await fetch(`${API_URL}/api/me`, {
    credentials: 'include',
  });
  if (res.status === 401) return null;
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return res.json();
}

export async function getGoogleLoginUrl(): Promise<{ url: string }> {
  if (MOCK_MODE) return (await import('./mock-api')).mockApi.getGoogleLoginUrl();
  return fetchApi('/api/auth/google/login');
}

export async function getGitHubLoginUrl(): Promise<{ url: string }> {
  if (MOCK_MODE) return (await import('./mock-api')).mockApi.getGitHubLoginUrl();
  return fetchApi('/api/auth/github/login');
}

export async function logout(): Promise<void> {
  if (MOCK_MODE) return (await import('./mock-api')).mockApi.logout();
  await fetch(`${API_URL}/api/auth/logout`, {
    method: 'POST',
    credentials: 'include',
  });
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
}

/** プロジェクトオーナーからのアップデート（モック/Phase5以降） */
export interface ProjectUpdate {
  id: string;
  project_id: string;
  created_at: string;
  title?: string | null;
  body: string;
  author_name?: string | null;
}

/** 金額表示タイプ: 希望額 / 必要額 / 両方 */
export type AmountInputType = 'want' | 'cost' | 'both';

export async function getProjects(limit = 20, offset = 0): Promise<Project[]> {
  if (MOCK_MODE) return (await import('./mock-api')).mockApi.getProjects(limit, offset);
  return fetchApi<Project[]>(`/api/projects?limit=${limit}&offset=${offset}`);
}

export async function getProject(id: string): Promise<Project> {
  if (MOCK_MODE) return (await import('./mock-api')).mockApi.getProject(id);
  return fetchApi<Project>(`/api/projects/${id}`);
}

export async function getMyProjects(): Promise<Project[]> {
  if (MOCK_MODE) return (await import('./mock-api')).mockApi.getMyProjects();
  return fetchApi<Project[]>('/api/me/projects');
}

/** 新着プロジェクト（モック時のみ、トップページ用） */
export async function getNewProjects(limit = 5): Promise<Project[]> {
  if (MOCK_MODE) return (await import('./mock-api')).mockApi.getNewProjects(limit);
  return getProjects(limit, 0);
}

/** HOT プロジェクト（モック時のみ、トップページ用） */
export async function getHotProjects(limit = 5): Promise<Project[]> {
  if (MOCK_MODE) return (await import('./mock-api')).mockApi.getHotProjects(limit);
  return getProjects(limit, 0);
}

// --- Activity Feed (モック時のみ) ---

export type ActivityType = 'project_created' | 'project_updated' | 'donation' | 'milestone';

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
  if (MOCK_MODE) return (await import('./mock-api')).mockApi.getActivityFeed(limit);
  return [];
}

// --- Project Chart (モック時のみ) ---

export interface ChartDataPoint {
  month: string;
  minAmount: number;
  targetAmount: number;
  actualAmount: number;
}

export async function getProjectChart(projectId: string): Promise<ChartDataPoint[]> {
  if (MOCK_MODE) return (await import('./mock-api')).mockApi.getProjectChart(projectId);
  return [];
}

/** プロジェクトオーナーからのアップデート（モック時のみ） */
export async function getProjectUpdates(projectId: string, limit = 20): Promise<ProjectUpdate[]> {
  if (MOCK_MODE) return (await import('./mock-api')).mockApi.getProjectUpdates(projectId, limit);
  return [];
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

export async function createProject(input: CreateProjectInput): Promise<Project> {
  if (MOCK_MODE) return (await import('./mock-api')).mockApi.createProject(input);
  return fetchApi<Project>('/api/projects', {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export interface UpdateProjectInput {
  name?: string;
  description?: string;
  deadline?: string | null;
  status?: string;
  owner_want_monthly?: number | null;
  costs?: ProjectCosts | null;
  alerts?: ProjectAlerts | null;
}

export async function updateProject(id: string, input: UpdateProjectInput): Promise<Project> {
  if (MOCK_MODE) return (await import('./mock-api')).mockApi.updateProject(id, input);
  return fetchApi<Project>(`/api/projects/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  });
}
