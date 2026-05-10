import type { User } from "@/entities/user/model/types";

import { apiClient } from "./client";

type BackendUser = {
  id: string;
  email: string;
  username?: string;
  first_name?: string;
  last_name?: string;
  role?: string;
};

const toFrontendUser = (payload: BackendUser): User => {
  const fullNameFromParts = `${payload.first_name ?? ""} ${payload.last_name ?? ""}`.trim();
  return {
    id: payload.id,
    email: payload.email,
    fullName: fullNameFromParts || payload.username || payload.email,
    role: payload.role === "admin" ? "admin" : "candidate",
    connectedGithub: false,
  };
};

export const userApi = {
  async getProfile(): Promise<User> {
    const { data } = await apiClient.get<BackendUser>("/users/me");
    return toFrontendUser(data);
  },

  async updateProfile(input: { fullName: string }): Promise<User> {
    const [firstName, ...rest] = input.fullName.trim().split(" ");
    const lastName = rest.join(" ");

    const { data } = await apiClient.put<BackendUser>("/users/me", {
      first_name: firstName || undefined,
      last_name: lastName || undefined,
      username: input.fullName,
    });

    return toFrontendUser(data);
  },

  /**
   * Rotate the password. 401 surfaces as a thrown error with status —
   * AuthForm-style consumers can match on response.status to render
   * 'invalid current password' inline.
   */
  async changePassword(currentPassword: string, newPassword: string): Promise<void> {
    await apiClient.put("/users/me/password", {
      current_password: currentPassword,
      new_password: newPassword,
    });
  },
};
