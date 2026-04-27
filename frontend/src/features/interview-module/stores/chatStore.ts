import { create } from "zustand";

import type { InterviewMessage } from "../types";

type ChatState = {
  messages: InterviewMessage[];
  pendingUserMessage: string;
  aiTyping: boolean;
  streamBuffer: string;
  setMessages: (messages: InterviewMessage[]) => void;
  addMessage: (message: InterviewMessage) => void;
  setPendingUserMessage: (value: string) => void;
  setAiTyping: (value: boolean) => void;
  pushStreamChunk: (chunk: string) => void;
  clearStreamBuffer: () => void;
  reset: () => void;
};

export const useChatStore = create<ChatState>((set) => ({
  messages: [],
  pendingUserMessage: "",
  aiTyping: false,
  streamBuffer: "",
  setMessages: (messages) => set({ messages }),
  addMessage: (message) => set((state) => ({ messages: [...state.messages, message] })),
  setPendingUserMessage: (pendingUserMessage) => set({ pendingUserMessage }),
  setAiTyping: (aiTyping) => set({ aiTyping }),
  pushStreamChunk: (chunk) => set((state) => ({ streamBuffer: state.streamBuffer + chunk })),
  clearStreamBuffer: () => set({ streamBuffer: "" }),
  reset: () => set({ messages: [], pendingUserMessage: "", aiTyping: false, streamBuffer: "" }),
}));
