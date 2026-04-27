import { apiClient } from "@/shared/api/client";
import type {
  CreateSessionPayload,
  CreateSessionResult,
  InterviewMessage,
  InterviewReport,
} from "./types";

type Wrapped<T> = {
  success?: boolean;
  data?: T;
  error?: string;
};

type SessionResponse = {
  session_id: string;
  ws_url: string;
  expires_at: string;
};

type SessionData = {
  session_id: string;
  ws_url?: string;
  status: "active" | "finished";
  role: string;
  level: string;
  started_at: string;
  expires_at: string;
  vacancy_title?: string;
  vacancy_category?: string;
  interview_mode?: string;
  focus_areas?: string[];
  primary_skills?: string[];
  theory_focus?: string[];
  practice_focus?: string[];
};

type MessageResponse = {
  messages: Array<{
    message_id: string;
    sender: "ai" | "user";
    content: string;
    topic?: string;
    difficulty?: number;
    created_at: string;
  }>;
};

type ReportResponse = {
  session_id: string;
  correctness: number;
  clarity: number;
  completeness: number;
  relevance: number;
  overall_score: number;
  strengths: string[];
  weaknesses: string[];
  recommendations: string[];
  generated_at: string;
};

const toMessage = (item: MessageResponse["messages"][number]): InterviewMessage => ({
  messageId: item.message_id,
  sender: item.sender,
  content: item.content,
  topic: item.topic,
  difficulty: item.difficulty,
  createdAt: item.created_at,
});

export const interviewModuleApi = {
  async createSession(payload: CreateSessionPayload): Promise<CreateSessionResult> {
    const { data } = await apiClient.post<Wrapped<SessionResponse>>("/interviews/sessions", {
      role: payload.role,
      level: payload.level,
      duration_minutes: payload.durationMinutes,
      question_limit: payload.questionLimit,
      vacancy_title: payload.vacancyTitle,
      vacancy_category: payload.vacancyCategory,
      interview_mode: payload.interviewMode,
      focus_areas: payload.focusAreas,
      primary_skills: payload.primarySkills,
      theory_focus: payload.theoryFocus,
      practice_focus: payload.practiceFocus,
    });

    if (!data.data) {
      throw new Error(data.error || "Не удалось создать сессию");
    }

    return {
      sessionId: data.data.session_id,
      wsUrl: data.data.ws_url,
      expiresAt: data.data.expires_at,
    };
  },

  async getSession(sessionId: string): Promise<SessionData> {
    const { data } = await apiClient.get<Wrapped<SessionData>>(`/interviews/sessions/${sessionId}`);
    if (!data.data) {
      throw new Error(data.error || "Сессия не найдена");
    }
    return data.data;
  },

  async getMessages(sessionId: string): Promise<InterviewMessage[]> {
    const { data } = await apiClient.get<Wrapped<MessageResponse>>(
      `/interviews/sessions/${sessionId}/messages`,
    );
    return (data.data?.messages || []).map(toMessage);
  },

  async sendMessage(sessionId: string, content: string): Promise<void> {
    await apiClient.post(`/interviews/sessions/${sessionId}/messages`, {
      content,
      client_message_id: crypto.randomUUID(),
    });
  },

  async finishSession(sessionId: string): Promise<void> {
    await apiClient.post(`/interviews/sessions/${sessionId}/finish`);
  },

  async getReport(sessionId: string): Promise<InterviewReport> {
    const { data } = await apiClient.get<Wrapped<ReportResponse>>(`/interviews/sessions/${sessionId}/report`);
    const report = data.data;
    if (!report) {
      throw new Error(data.error || "Отчет недоступен");
    }

    return {
      sessionId: report.session_id,
      correctness: report.correctness,
      clarity: report.clarity,
      completeness: report.completeness,
      relevance: report.relevance,
      overallScore: report.overall_score,
      strengths: report.strengths,
      weaknesses: report.weaknesses,
      recommendations: report.recommendations,
      generatedAt: report.generated_at,
    };
  },
};
