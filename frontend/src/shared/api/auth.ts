import { apiClient } from "./client";

type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

type TokenPayload = {
  access_token?: string;
  refresh_token?: string;
  accessToken?: string;
  refreshToken?: string;
};

const mapTokens = (payload: TokenPayload): AuthTokens => ({
  accessToken: payload.access_token ?? payload.accessToken ?? "",
  refreshToken: payload.refresh_token ?? payload.refreshToken ?? "",
});

export const authApi = {
  async login(email: string, password: string): Promise<AuthTokens> {
    const { data } = await apiClient.post<TokenPayload>("/auth/login", { email, password });
    return mapTokens(data);
  },

  async register(email: string, password: string, fullName: string): Promise<AuthTokens> {
    const { data } = await apiClient.post<TokenPayload>("/auth/register", {
      email,
      password,
      username: fullName,
    });
    return mapTokens(data);
  },

  async refresh(refreshToken: string): Promise<AuthTokens> {
    const { data } = await apiClient.post<TokenPayload>("/auth/refresh", {
      refresh_token: refreshToken,
    });
    return mapTokens(data);
  },
};
