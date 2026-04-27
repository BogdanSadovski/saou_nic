export type InterviewRole =
  | "Frontend"
  | "Backend"
  | "Web"
  | "Mobile"
  | "Data"
  | "ML"
  | "DevOps"
  | "Game"
  | "Security"
  | "Systems"
  | "Enterprise"
  | "Fintech"
  | "IoT"
  | "Management";
export type InterviewLevel = "Junior" | "Middle" | "Senior";
export type InterviewMode = "practice" | "theory";

export type VacancyOption = {
  id: string;
  title: string;
  category: InterviewRole;
  description: string;
  searchTerms: string[];
  focusAreas: string[];
  primarySkills: string[];
  theoryFocus: string[];
  practiceFocus: string[];
};

export type SessionStatus = "idle" | "active" | "finished";

export type InterviewSession = {
  sessionId: string;
  status: SessionStatus;
  role: InterviewRole;
  level: InterviewLevel;
  vacancyTitle?: string;
  vacancyCategory?: InterviewRole;
  interviewMode?: InterviewMode;
  focusAreas?: string[];
  primarySkills?: string[];
  theoryFocus?: string[];
  practiceFocus?: string[];
  startedAt: string;
  endsAt: string;
};

export type InterviewMessage = {
  messageId: string;
  sender: "ai" | "user";
  content: string;
  topic?: string;
  difficulty?: number;
  createdAt: string;
};

export type CreateSessionPayload = {
  role: InterviewRole;
  level: InterviewLevel;
  durationMinutes: number;
  questionLimit: number;
  vacancyTitle: string;
  vacancyCategory: InterviewRole;
  interviewMode: InterviewMode;
  focusAreas: string[];
  primarySkills: string[];
  theoryFocus: string[];
  practiceFocus: string[];
};

export type CreateSessionResult = {
  sessionId: string;
  wsUrl: string;
  expiresAt: string;
};

export type InterviewReport = {
  sessionId: string;
  correctness: number;
  clarity: number;
  completeness: number;
  relevance: number;
  overallScore: number;
  strengths: string[];
  weaknesses: string[];
  recommendations: string[];
  generatedAt: string;
};
