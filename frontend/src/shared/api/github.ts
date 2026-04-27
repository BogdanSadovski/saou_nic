import { apiClient } from "./client";

export type GithubChartPoint = {
  label: string;
  value: number;
};

export type GithubContributionDay = {
  date: string;
  count: number;
};

export type GithubTopRepository = {
  name: string;
  url: string;
  description?: string;
  language?: string;
  stars: number;
  forks: number;
  open_issues: number;
  last_push?: string;
};

export type GithubRoleRecommendation = {
  role: string;
  fit_score: number;
  rationale: string;
};

export type GithubLanguageInsight = {
  language: string;
  confidence: number;
  evidence: string;
  interview_topics: string[];
};

export type GithubInterviewTrack = {
  role: string;
  mode: string;
  level: string;
  duration_minutes: number;
  focus_areas: string[];
  primary_skills: string[];
  rationale: string;
};

export type GithubImportResponse = {
  username: string;
  profile_url: string;
  profile_name?: string;
  bio?: string;
  avatar_url?: string;
  stats: {
    followers: number;
    following: number;
    public_repos: number;
    sampled_repos: number;
    total_stars: number;
    total_forks: number;
    total_open_issues: number;
  };
  charts: {
    language_distribution: GithubChartPoint[];
    monthly_activity: GithubChartPoint[];
    contribution_days: GithubContributionDay[];
  };
  top_repositories: GithubTopRepository[];
  ai_insights: {
    summary: string;
    strengths: string[];
    risks: string[];
    action_plan: string[];
    language_insights: GithubLanguageInsight[];
    interview_tracks: GithubInterviewTrack[];
    recommended_positions: GithubRoleRecommendation[];
  };
};

type WrappedResponse<T> = {
  success?: boolean;
  data?: T;
  error?: string;
};

const isObject = (value: unknown): value is Record<string, unknown> => {
  return typeof value === "object" && value !== null;
};

const toErrorMessage = (error: unknown, fallback: string): string => {
  if (isObject(error) && "response" in error) {
    const response = (error as { response?: { data?: unknown; status?: number } }).response;
    if (response && isObject(response.data) && typeof response.data.error === "string") {
      return response.data.error;
    }
    if (response?.status === 404) {
      return "Эндпоинт импорта не найден (404). Проверьте маршрут API.";
    }
  }
  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }
  return fallback;
};

export const githubApi = {
  async importProfile(payload: {
    profileUrl: string;
    maxRepos?: number;
    rolePreferences?: string[];
  }): Promise<GithubImportResponse> {
    const requestBody = {
      profile_url: payload.profileUrl,
      max_repos: payload.maxRepos,
      role_preferences: payload.rolePreferences,
    };

    let data: WrappedResponse<GithubImportResponse> | undefined;
    try {
      const response = await apiClient.post<WrappedResponse<GithubImportResponse>>("/github/import", requestBody);
      data = response.data;
    } catch (error) {
      const status = isObject(error) && "response" in error
        ? (error as { response?: { status?: number } }).response?.status
        : undefined;

      if (status === 404) {
        try {
          const fallback = await apiClient.post<WrappedResponse<GithubImportResponse>>("/v1/github/import", requestBody);
          data = fallback.data;
        } catch (fallbackError) {
          throw new Error(toErrorMessage(fallbackError, "Не удалось импортировать GitHub-профиль"));
        }
      } else {
        throw new Error(toErrorMessage(error, "Не удалось импортировать GitHub-профиль"));
      }
    }

    if (!data?.data) {
      throw new Error(data?.error || "Не удалось импортировать GitHub-профиль");
    }

    return data.data;
  },
};
