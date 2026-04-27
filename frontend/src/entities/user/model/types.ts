export type User = {
  id: string;
  fullName: string;
  email: string;
  role: "candidate" | "admin";
  avatarUrl?: string;
  connectedGithub: boolean;
};
