import type { PermissionNode } from "../services/contracts";

export const menuNames: Record<string, string> = {
  "/lead-popup": "留资弹窗配置",
  "/leads": "留资记录",
  "/faqs": "问答配置",
  "/media": "媒体库",
  "/categories": "商品分类",
  "/products": "商品管理",
  "/whatsapp": "客服配置",
  "/articles": "新闻管理",
  "/settings": "品牌与主题",
  "/banners": "Banner管理",
  "/system/roles": "角色权限管理",
  "/system/users": "用户管理",
  "/system/security": "个人中心",
  "/system/audit": "操作日志",
  "/system/config": "运行配置状态",
};

const groups = [
  { code: "navigation.operations", name: "运营管理", routes: ["/media", "/categories", "/products", "/whatsapp", "/articles", "/faqs", "/leads"] },
  { code: "navigation.settings", name: "系统配置", routes: ["/settings", "/banners", "/lead-popup"] },
  { code: "navigation.users", name: "用户管理", routes: ["/system/roles", "/system/users", "/system/security"] },
  { code: "navigation.maintenance", name: "系统维护", routes: ["/system/audit", "/system/config"] },
];

// Only regroup nodes supplied by the server; never create access to a page.
export function groupPermissionMenus(nodes: PermissionNode[]): PermissionNode[] {
  const pages: PermissionNode[] = [];
  function collect(items: PermissionNode[]) {
    for (const node of items) {
      if (node.type === "directory") collect(node.children);
      else if (node.type === "menu" && node.route !== "/overview") {
        pages.push({ ...node, name: menuNames[node.route || ""] || node.name });
      }
    }
  }
  collect(nodes);
  const assigned = new Set<string>();
  const result: PermissionNode[] = [];
  for (const [sort, group] of groups.entries()) {
    const children = group.routes.flatMap((route) => pages.filter((node) => node.route === route));
    if (!children.length) continue;
    children.forEach((node) => assigned.add(node.code));
    result.push({ code: group.code, name: group.name, type: "directory", route: null, permissionCode: null, sort, children });
  }
  return [...result, ...pages.filter((node) => !assigned.has(node.code))];
}
