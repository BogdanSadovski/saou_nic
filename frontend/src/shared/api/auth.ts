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

/**
 * Map a backend token payload to our shape and reject any response that
 * is missing a usable access token. Previously this returned `""` for
 * missing tokens and the auth store stored the empty string as if the
 * user were authenticated, leading to silent 401 storms downstream.
 */
const mapTokens = (payload: TokenPayload | null | undefined): AuthTokens => {
  const accessToken = payload?.access_token ?? payload?.accessToken ?? "";
  const refreshToken = payload?.refresh_token ?? payload?.refreshToken ?? "";
  if (!accessToken) {
    throw new Error("Сервер не вернул токен авторизации");
  }
  return { accessToken, refreshToken };
};

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
