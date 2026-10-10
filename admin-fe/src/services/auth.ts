import { api } from "./cms";
import type { PermissionNode } from "./contracts";
export interface User {
  id: string;
  username: string;
  displayName: string;
  roleId: string;
  enabled: boolean;
  mustChangePwd: boolean;
  totpBound: boolean;
  createdAt: string;
  updatedAt: string;
}
export interface Role {
  id: string;
  code: string;
  name: string;
  enabled: boolean;
  isSuper: boolean;
}
export interface Identity {
  user: User;
  role: Role;
  scope: "full" | "onboarding" | "password_change";
  permissions: string[];
  menu: PermissionNode[];
}
export interface InitialState {
  identity: Identity | null;
}
export const me = () => api<Identity>("me");
export const allows = (
  identity: Identity | null | undefined,
  permission: string,
) => identity?.scope === "full" && identity.permissions.includes(permission);
export async function allRoles() {
  const rows: Role[] = [];
  for (let page = 1; ; page++) {
    const result = await api<{ items: Role[]; total: number }>(
      `roles?page=${page}&pageSize=100`,
    );
    rows.push(...result.items);
    if (rows.length >= result.total || !result.items.length) return rows;
  }
}
