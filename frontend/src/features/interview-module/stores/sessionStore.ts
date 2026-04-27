import { create } from "zustand";

import type { InterviewLevel, InterviewMode, InterviewRole, SessionStatus } from "../types";

type SessionState = {
  sessionId: string | null;
  status: SessionStatus;
  role: InterviewRole;
  level: InterviewLevel;
  vacancyTitle: string;
  vacancyCategory: InterviewRole;
  interviewMode: InterviewMode;
  focusAreas: string[];
  primarySkills: string[];
  theoryFocus: string[];
  practiceFocus: string[];
  startedAt: string | null;
  endsAt: string | null;
  setSession: (payload: {
    sessionId: string;
    role: InterviewRole;
    level: InterviewLevel;
    vacancyTitle: string;
    vacancyCategory: InterviewRole;
    interviewMode: InterviewMode;
    focusAreas: string[];
    primarySkills: string[];
    theoryFocus: string[];
    practiceFocus: string[];
    startedAt: string;
    endsAt: string;
  }) => void;
  setStatus: (status: SessionStatus) => void;
  reset: () => void;
};

const initialState = {
  sessionId: null,
  status: "idle" as SessionStatus,
  role: "Backend" as InterviewRole,
  level: "Middle" as InterviewLevel,
  vacancyTitle: "Backend Engineer",
  vacancyCategory: "Backend" as InterviewRole,
  interviewMode: "practice" as InterviewMode,
  focusAreas: [] as string[],
  primarySkills: [] as string[],
  theoryFocus: [] as string[],
  practiceFocus: [] as string[],
  startedAt: null,
  endsAt: null,
};

export const useSessionStore = create<SessionState>((set) => ({
  ...initialState,
  setSession: ({
    sessionId,
    role,
    level,
    vacancyTitle,
    vacancyCategory,
    interviewMode,
    focusAreas,
    primarySkills,
    theoryFocus,
    practiceFocus,
    startedAt,
    endsAt,
  }) => {
    set({
      sessionId,
      role,
      level,
      vacancyTitle,
      vacancyCategory,
      interviewMode,
      focusAreas,
      primarySkills,
      theoryFocus,
      practiceFocus,
      startedAt,
      endsAt,
      status: "active",
    });
  },
  setStatus: (status) => set({ status }),
  reset: () => set(initialState),
}));
