import type { ReportSummary } from "@/entities/report/model/types";

import { apiClient } from "./client";

export type UserInterviewEntry = {
  session_id: string;
  role: string;
  level: string;
  vacancy_title?: string;
  interview_mode: string;
  status: string;
  current_topic?: string;
  duration_minutes: number;
  question_limit: number;
  messages_total: number;
  ai_messages: number;
  user_messages: number;
  started_at: string;
  expires_at: string;
  finished_at?: string;
  overall_score?: number;
  strengths?: string[];
  weaknesses?: string[];
};

export type UserInterviewAnalyticsReport = {
  user_id: string;
  generated_at: string;
  totals: {
    total_interviews: number;
    completed_interviews: number;
    in_progress_interviews: number;
    expired_interviews: number;
    completion_rate: number;
  };
  performance: {
    average_score: number;
    best_score: number;
    latest_score: number;
    reports_generated: number;
    avg_question_count: number;
    avg_session_minutes: number;
  };
  role_distribution: Array<{ label: string; value: number }>;
  mode_distribution: Array<{ label: string; value: number }>;
  timeline: Array<{ date: string; started: number; completed: number }>;
  top_strengths: string[];
  top_weaknesses: string[];
  top_recommendations: string[];
  completed_interviews: UserInterviewEntry[];
  incomplete_interviews: UserInterviewEntry[];
  recent_interviews: UserInterviewEntry[];
};

type BackendReport = {
  id: string;
  title?: string;
  candidate_name?: string;
  overall_score?: number;
  recommendation?: string;
  created_at?: string;
};

type WrappedResponse<T> = {
  data?: T;
};

const mapRecommendation = (value: string | undefined): "hire" | "consider" | "reject" => {
  if (!value) return "consider";
  const normalized = value.toLowerCase();
  if (normalized.includes("hire")) return "hire";
  if (normalized.includes("reject")) return "reject";
  return "consider";
};

export const reportsApi = {
  async listReports(): Promise<ReportSummary[]> {
    const { data } = await apiClient.get<WrappedResponse<BackendReport[]>>("/reports");
    const reports = data.data ?? [];
    return reports.map((item) => ({
      id: item.id,
      candidateName: item.candidate_name ?? item.title ?? "Candidate",
      overallScore: item.overall_score ?? 0,
      recommendation: mapRecommendation(item.recommendation),
      createdAt: item.created_at ?? new Date().toISOString(),
    }));
  },

  async getMyInterviewReport(): Promise<UserInterviewAnalyticsReport> {
    const { data } = await apiClient.get<WrappedResponse<UserInterviewAnalyticsReport>>("/interviews/me/report");
    if (!data.data) {
      throw new Error("Не удалось получить отчет пользователя");
    }

    const payload = data.data;
    return {
      ...payload,
      role_distribution: payload.role_distribution ?? [],
      mode_distribution: payload.mode_distribution ?? [],
      timeline: payload.timeline ?? [],
      top_strengths: payload.top_strengths ?? [],
      top_weaknesses: payload.top_weaknesses ?? [],
      top_recommendations: payload.top_recommendations ?? [],
      completed_interviews: payload.completed_interviews ?? [],
      incomplete_interviews: payload.incomplete_interviews ?? [],
      recent_interviews: payload.recent_interviews ?? [],
    };
  },

  /**
   * Build an empty analytics report skeleton. Used by UI as a graceful
   * fallback when the backend is unreachable or the report endpoint
   * returns 404, so the page can still render its shell, search and
   * "create your first interview" CTA instead of an error wall.
   */
  emptyReport(userId = "anonymous"): UserInterviewAnalyticsReport {
    return {
      user_id: userId,
      generated_at: new Date().toISOString(),
      totals: {
        total_interviews: 0,
        completed_interviews: 0,
        in_progress_interviews: 0,
        expired_interviews: 0,
        completion_rate: 0,
      },
      performance: {
        average_score: 0,
        best_score: 0,
        latest_score: 0,
        reports_generated: 0,
        avg_question_count: 0,
        avg_session_minutes: 0,
      },
      role_distribution: [],
      mode_distribution: [],
      timeline: [],
      top_strengths: [],
      top_weaknesses: [],
      top_recommendations: [],
      completed_interviews: [],
      incomplete_interviews: [],
      recent_interviews: [],
    };
  },
};
