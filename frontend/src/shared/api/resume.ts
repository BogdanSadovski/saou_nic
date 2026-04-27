import { apiClient } from "./client";

export type ResumeChartPoint = {
  label: string;
  value: number;
};

export type ResumeRoleRecommendation = {
  role: string;
  fit_score: number;
  rationale: string;
};

export type ResumeLanguageInsight = {
  language: string;
  confidence: number;
  evidence: string;
  interview_topics: string[];
};

export type ResumeInterviewTrack = {
  role: string;
  mode: string;
  level: string;
  duration_minutes: number;
  focus_areas: string[];
  primary_skills: string[];
  rationale: string;
};

export type ResumeImportResponse = {
  report_id: string;
  created_at: string;
  file_name: string;
  file_size: number;
  content_type: string;
  detected_format: string;
  processing_stages: Array<{
    code: string;
    title: string;
    status: string;
    duration_ms: number;
  }>;
  stats: {
    word_count: number;
    character_count: number;
    estimated_pages: number;
    skills_count: number;
    language_count: number;
    experience_entries: number;
    education_entries: number;
  };
  charts: {
    language_distribution: ResumeChartPoint[];
    skills_distribution: ResumeChartPoint[];
  };
  extracted_skills: string[];
  ai_insights: {
    summary: string;
    strong_points: string[];
    improvement_points: string[];
    action_plan: string[];
    language_insights: ResumeLanguageInsight[];
    interview_tracks: ResumeInterviewTrack[];
    recommended_positions: ResumeRoleRecommendation[];
  };
};

type ResumeHistoryResponse = {
  items: ResumeImportResponse[];
  total: number;
};

type WrappedResponse<T> = {
  data?: T;
  success?: boolean;
  error?: string;
};

export const resumeApi = {
  async uploadResume(
    file: File,
    rolePreferences: string[] = [],
    onUploadProgress?: (percent: number) => void,
  ): Promise<ResumeImportResponse> {
    const form = new FormData();
    form.append("file", file);
    if (rolePreferences.length > 0) {
      form.append("role_preferences", rolePreferences.join(","));
    }

    let data: WrappedResponse<ResumeImportResponse> | undefined;
    const uploadConfig = {
      headers: {
        "Content-Type": "multipart/form-data",
      },
      timeout: 120_000,
      onUploadProgress: (event: { total?: number; loaded: number }) => {
        if (!event.total || !onUploadProgress) {
          return;
        }
        const percent = Math.max(1, Math.min(100, Math.round((event.loaded / event.total) * 100)));
        onUploadProgress(percent);
      },
    };

    try {
      const response = await apiClient.post<WrappedResponse<ResumeImportResponse>>("/resume/import", form, uploadConfig);
      data = response.data;
    } catch (error) {
      const status = (error as { response?: { status?: number } })?.response?.status;
      if (status === 404) {
        try {
          const fallback = await apiClient.post<WrappedResponse<ResumeImportResponse>>("/v1/resume/import", form, uploadConfig);
          data = fallback.data;
        } catch {
          throw new Error("Сервис импорта резюме недоступен (404). Проверьте, что api-gateway обновлен и перезапущен.");
        }
      } else {
        const code = (error as { code?: string })?.code;
        if (code === "ECONNABORTED") {
          throw new Error("Сервер обрабатывает резюме слишком долго. Попробуйте файл меньшего размера или повторите попытку чуть позже.");
        }
        const message =
          (error as { response?: { data?: { error?: string; message?: string } } })?.response?.data?.error ||
          (error as { response?: { data?: { error?: string; message?: string } } })?.response?.data?.message ||
          "Не удалось загрузить и проанализировать резюме";
        throw new Error(message);
      }
    }

    if (!data?.data) {
      throw new Error(data?.error || "Сервис импорта резюме вернул пустой ответ");
    }

    return data.data;
  },

  async getHistory(): Promise<ResumeImportResponse[]> {
    try {
      const response = await apiClient.get<WrappedResponse<ResumeHistoryResponse>>("/resume/history");
      return response.data?.data?.items || [];
    } catch (error) {
      const status = (error as { response?: { status?: number } })?.response?.status;
      if (status === 404) {
        try {
          const fallback = await apiClient.get<WrappedResponse<ResumeHistoryResponse>>("/v1/resume/history");
          return fallback.data?.data?.items || [];
        } catch {
          throw new Error("История резюме временно недоступна (404). Обновите api-gateway.");
        }
      }
      throw error;
    }
  },

  async getReport(reportID: string): Promise<ResumeImportResponse> {
    try {
      const response = await apiClient.get<WrappedResponse<ResumeImportResponse>>(`/resume/history/${reportID}`);
      if (!response.data?.data) {
        throw new Error("Отчет не найден");
      }
      return response.data.data;
    } catch (error) {
      const status = (error as { response?: { status?: number } })?.response?.status;
      if (status === 404) {
        try {
          const fallback = await apiClient.get<WrappedResponse<ResumeImportResponse>>(`/v1/resume/history/${reportID}`);
          if (!fallback.data?.data) {
            throw new Error("Отчет не найден");
          }
          return fallback.data.data;
        } catch {
          throw new Error("Отчет резюме недоступен (404). Проверьте обновление API-маршрутов.");
        }
      }
      throw error;
    }
  },
};
