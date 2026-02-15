import type {
  Project,
  ProjectCosts,
  ProjectAlerts,
  CreateProjectInput,
  UpdateProjectInput,
  User,
} from './api';
import type { ProjectUpdate } from './api';
import { MOCK_PROJECTS, type MockProject } from '../data/mock-projects';
import { MOCK_ACTIVITIES, MOCK_OWNERS, MOCK_RECENT_SUPPORTERS, type ActivityItem } from '../data/mock-activities';
import { MOCK_CHART_DATA, type ChartDataPoint } from '../data/mock-chart-data';
import { MOCK_PROJECT_UPDATES } from '../data/mock-project-updates';

const delay = (ms: number) => new Promise((r) => setTimeout(r, ms));

/** モック用の擬似遅延（体感用、0 にしても可） */
const MOCK_DELAY = 150;

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
    return null;
  },

  async getGoogleLoginUrl(): Promise<{ url: string }> {
    await delay(MOCK_DELAY);
    return { url: '#' };
  },

  async getGitHubLoginUrl(): Promise<{ url: string }> {
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
    return toProject(p);
  },

  async getMyProjects(): Promise<Project[]> {
    await delay(MOCK_DELAY);
    return [];
  },

  async createProject(input: CreateProjectInput): Promise<Project> {
    await delay(MOCK_DELAY);
    const id = `mock-new-${Date.now()}`;
    const now = new Date().toISOString();
    const newProject: Project = {
      id,
      owner_id: 'user-mock',
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
    return toProject({
      ...p,
      ...input,
      id: p.id,
      owner_id: p.owner_id,
      created_at: p.created_at,
    });
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

  /** プロジェクトオーナーからのアップデート */
  async getProjectUpdates(projectId: string, limit = 20): Promise<ProjectUpdate[]> {
    await delay(MOCK_DELAY);
    const list = MOCK_PROJECT_UPDATES[projectId] ?? [];
    return list
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
      .slice(0, limit);
  },
};
