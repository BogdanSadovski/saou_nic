export type InterviewQuestion = {
  id: string;
  text: string;
  category: "system-design" | "algorithms" | "behavioral";
  isActive?: boolean;
};

export type InterviewMessage = {
  id: string;
  role: "ai" | "user";
  content: string;
  timestamp: string;
};
