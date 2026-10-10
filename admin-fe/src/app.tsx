import "@shop/content/styles.css";
import { me, type InitialState } from "./services/auth";
import type { PermissionNode } from "./services/contracts";
export async function getInitialState(): Promise<InitialState> {
  try {
    return { identity: await me() };
  } catch {
    return { identity: null };
  }
}
function menus(nodes: PermissionNode[]): unknown[] {
  return nodes
    .filter((node) => node.type !== "action")
    .map((node) => ({
      key: node.code,
      name: node.name,
      path: node.route || undefined,
      children: node.children.length ? menus(node.children) : undefined,
    }));
}
export const layout = ({ initialState }: { initialState?: InitialState }) => ({
  logo: false,
  menu: { locale: false },
  menuDataRender: () => menus(initialState?.identity?.menu || []),
});
