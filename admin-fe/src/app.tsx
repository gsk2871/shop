import "./global.css";
import {
  AppstoreOutlined,
  FileTextOutlined,
  PictureOutlined,
  SettingOutlined,
  DashboardOutlined,
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
export async function getInitialState(): Promise<InitialState> {
  try {
    return { identity: await me() };
  } catch {
    return { identity: null };
  }
}
const menuIcons: Record<string, ReactNode> = {
  "/overview": <DashboardOutlined />,
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
  return nodes
    .filter((node) => node.type !== "action")
    .flatMap((node) =>
      node.type === "directory"
        ? menus(node.children)
        : [
            {
              key: node.code,
              name: node.route === "/articles" ? "新闻管理" : node.name,
              icon: menuIcons[node.route || ""],
              path: node.route || undefined,
              children: node.children.length ? menus(node.children) : undefined,
            },
          ],
    );
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
        "安全中心"}
    </Button>,
  ],
  menu: { locale: false },
  menuDataRender: () => menus(initialState?.identity?.menu || []),
});
