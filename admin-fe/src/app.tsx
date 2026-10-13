import "./global.css";
import {
  AppstoreOutlined,
  FileTextOutlined,
  PictureOutlined,
  SettingOutlined,
  TeamOutlined,
  SafetyOutlined,
  AuditOutlined,
  CustomerServiceOutlined,
  TagsOutlined,
  LayoutOutlined,
} from "@ant-design/icons";
import { history } from "@umijs/max";
import { Button } from "antd";
import type { ReactNode } from "react";
import "@shop/content/styles.css";
import { me, type InitialState } from "./services/auth";
import type { PermissionNode } from "./services/contracts";
import { groupPermissionMenus } from "./navigation";
export async function getInitialState(): Promise<InitialState> {
  try {
    return { identity: await me() };
  } catch {
    return { identity: null };
  }
}
const menuIcons: Record<string, ReactNode> = {
  "navigation.operations": <AppstoreOutlined />,
  "navigation.settings": <SettingOutlined />,
  "navigation.users": <TeamOutlined />,
  "navigation.maintenance": <AuditOutlined />,
  "/products": <AppstoreOutlined />,
  "/categories": <TagsOutlined />,
  "/articles": <FileTextOutlined />,
  "/media": <PictureOutlined />,
  "/banners": <LayoutOutlined />,
  "/settings": <SettingOutlined />,
  "/whatsapp": <CustomerServiceOutlined />,
  "/system/users": <TeamOutlined />,
  "/system/roles": <SafetyOutlined />,
  "/system/security": <SafetyOutlined />,
  "/system/audit": <AuditOutlined />,
  "/system/config": <SettingOutlined />,
};
function menus(nodes: PermissionNode[]): unknown[] {
  return nodes.filter((node) => node.type !== "action").map((node) => ({
    key: node.code,
    name: node.name,
    icon: menuIcons[node.route || node.code],
    path: node.route || undefined,
    children: node.children.some((child) => child.type !== "action")
      ? menus(node.children)
      : undefined,
  }));
}
export const layout = ({ initialState }: { initialState?: InitialState }) => ({
  title: "Shop Admin",
  logo: <span className="admin-brand-mark">S</span>,
  navTheme: "light" as const,
  layout: "mix" as const,
  contentWidth: "Fluid" as const,
  fixSiderbar: true,
  colorPrimary: "#1677ff",
  token: {
    pageContainer: { paddingInlinePageContainerContent: 24 },
    sider: {
      colorMenuBackground: "#fff",
      colorBgMenuItemSelected: "#e6f4ff",
      colorTextMenuSelected: "#1677ff",
    },
  },
  actionsRender: () => [
    <Button
      key="account"
      type="text"
      icon={<SafetyOutlined />}
      onClick={() => history.push("/system/security")}
    >
      {initialState?.identity?.user.displayName ||
        initialState?.identity?.user.username ||
        "个人中心"}
    </Button>,
  ],
  menu: { locale: false },
  menuDataRender: () => menus(groupPermissionMenus(initialState?.identity?.menu || [])),
});
