import { api } from "./cms";
export interface ConfigStatus {
  version: number;
  loadedAt: string;
  checkedAt: string;
  error?: string;
  sources: Record<string, string>;
}
export const configStatus = () => api<ConfigStatus>("config/status");
