export type ReportSummary = {
  id: string;
  candidateName: string;
  overallScore: number;
  recommendation: "hire" | "consider" | "reject";
  createdAt: string;
};
