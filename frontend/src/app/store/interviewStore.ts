import { create } from "zustand";

import type {
  InterviewMessage,
  InterviewQuestion,
} from "@/entities/interview/model/types";
import { interviewApi } from "@/shared/api";

type InterviewState = {
  questions: InterviewQuestion[];
  messages: InterviewMessage[];
  elapsedSec: number;
  isPaused: boolean;
  loadQuestions: () => Promise<void>;
  setPaused: (paused: boolean) => void;
  setActiveQuestion: (questionId: string) => void;
  addMessage: (message: InterviewMessage) => void;
  tick: () => void;
  reset: () => void;
};

const starterQuestions: InterviewQuestion[] = [
  {
    id: "q1",
    text: "Спроектируйте отказоустойчивый пайплайн уведомлений для 1 млн пользователей.",
    category: "system-design",
    isActive: true,
  },
  {
    id: "q2",
    text: "Как оптимизировать поиск top-K при динамических обновлениях?",
    category: "algorithms",
  },
  {
    id: "q3",
    text: "Опишите сложный продакшн-инцидент и вашу роль в его устранении.",
    category: "behavioral",
  },
];

export const useInterviewStore = create<InterviewState>((set) => ({
  questions: starterQuestions,
  messages: [
    {
      id: "m1",
      role: "ai",
      content: "Здравствуйте. Я ваш ИИ-интервьюер. Начнем с архитектуры.",
      timestamp: new Date().toISOString(),
    },
  ],
  elapsedSec: 0,
  isPaused: false,
  loadQuestions: async () => {
    try {
      const questions = await interviewApi.listQuestions();
      if (questions.length > 0) {
        set({
          questions: questions.map((question, index) => ({
            ...question,
            isActive: index === 0,
          })),
        });
      }
    } catch {
      // Keep local fallback questions when backend data isn't available.
    }
  },
  setPaused: (paused) => set({ isPaused: paused }),
  setActiveQuestion: (questionId) =>
    set((state) => ({
      questions: state.questions.map((question) => ({
        ...question,
        isActive: question.id === questionId,
      })),
    })),
  addMessage: (message) =>
    set((state) => ({
      messages: [...state.messages, message],
    })),
  tick: () =>
    set((state) => ({
      elapsedSec: state.isPaused ? state.elapsedSec : state.elapsedSec + 1,
    })),
  reset: () =>
    set({
      questions: starterQuestions,
      messages: [
        {
          id: "m1",
          role: "ai",
          content: "Сессия перезапущена. Расскажите о вашем самом сильном проекте.",
          timestamp: new Date().toISOString(),
        },
      ],
      elapsedSec: 0,
      isPaused: false,
    }),
}));
