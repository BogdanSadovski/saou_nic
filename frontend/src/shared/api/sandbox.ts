import { apiClient } from "./client";

/**
 * Code sandbox API — REST-обёртка над services/code-executor.
 *
 * Бэк выполняет код в одноразовом docker-контейнере с лимитами
 * (5-45с wall-time в зависимости от языка, 128m-512m RAM, no network).
 * Для SQL — in-memory SQLite с предзаряженной схемой users/orders.
 */

export type SandboxLanguageInfo = {
  id: string;
  label: string;
  in_process: boolean;
};

export type SandboxLanguagesResponse = {
  languages: SandboxLanguageInfo[];
  limits: {
    wall_timeout_sec: number;
    memory: string;
    cpu: string;
    output_byte_cap: number;
  };
};

export type SandboxExecuteRequest = {
  language: string;
  code: string;
  stdin?: string;
};

export type SandboxExecuteResponse = {
  language: string;
  stdout: string;
  stderr: string;
  exit_code: number;
  duration_ms: number;
  timed_out: boolean;
  error?: string | null;
};

export const sandboxApi = {
  async listLanguages(): Promise<SandboxLanguagesResponse> {
    const { data } = await apiClient.get<SandboxLanguagesResponse>("/sandbox/languages");
    return data;
  },

  async execute(req: SandboxExecuteRequest): Promise<SandboxExecuteResponse> {
    const { data } = await apiClient.post<SandboxExecuteResponse>("/sandbox/execute", req);
    return data;
  },
};
