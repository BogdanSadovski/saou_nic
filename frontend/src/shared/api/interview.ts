import type { InterviewQuestion } from "@/entities/interview/model/types";

import { apiClient } from "./client";

type InterviewResponse = {
  id: string;
  title?: string;
  description?: string;
  questions?: Array<{ id: string; title?: string; description?: string; type?: string }>;
};

type WrappedResponse<T> = {
  data?: T;
  success?: boolean;
};

const mapQuestion = (q: { id: string; title?: string; description?: string; type?: string }): InterviewQuestion => ({
  id: q.id,
  text: q.title || q.description || "Вопрос",
  category: q.type === "behavioral" ? "behavioral" : q.type === "coding" ? "algorithms" : "system-design",
});

export const interviewApi = {
  async listQuestions(): Promise<InterviewQuestion[]> {
    const { data } = await apiClient.get<WrappedResponse<InterviewResponse[]>>("/interviews");
    const interviews = data.data ?? [];
    const first = interviews[0];
    if (!first || !first.questions) {
      return [];
    }
    return first.questions.map(mapQuestion);
  },

  async analyzeAnswer(answer: string): Promise<string> {
    const { data } = await apiClient.post<{ feedback?: string }>("/ai/analysis/answer", {
      question: "Вопрос интервью",
      answer,
      expected_answer: "Структурированный и краткий ответ с разбором компромиссов",
      rubric: "Ясность, корректность, компромиссы",
    });

    return data.feedback ?? "Ответ получен и проанализирован.";
  },
};
