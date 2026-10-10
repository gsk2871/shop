import { defineConfig } from "@umijs/max";

export default defineConfig({
  title: "管理后台",
  hash: true,
  esbuildMinifyIIFE: true,
  antd: {},
  layout: { title: "管理后台" },
  request: {},
  initialState: {},
  model: {},
  routes: [
    { path: "/categories", name: "商品分类", component: "./Categories", wrappers: ["@/components/Auth"] },
    {
      path: "/whatsapp",
      name: "WhatsApp 客服配置",
      component: "./WhatsApp",
      wrappers: ["@/components/Auth"],
    },
    {
      path: "/banners",
      name: "Banner 投放",
      component: "./Banners",
      wrappers: ["@/components/Auth"],
    },
    {
      path: "/system/users",
      name: "用户管理",
      component: "./Users",
      wrappers: ["@/components/Auth"],
    },
    {
      path: "/system/roles",
      name: "角色管理",
      component: "./Roles",
      wrappers: ["@/components/Auth"],
    },
    {
      path: "/system/audit",
      name: "操作日志",
      component: "./Audit",
      wrappers: ["@/components/Auth"],
    },
    {
      path: "/system/security",
      name: "个人安全中心",
      component: "./Security",
      wrappers: ["@/components/Auth"],
    },
    {
      path: "/system/config",
      name: "运行配置",
      component: "./Config",
      wrappers: ["@/components/Auth"],
    },
    { path: "/login", component: "./Login", layout: false },
    { path: "/", redirect: "/overview" },
    {
      path: "/overview",
      name: "工作台",
      component: "./Overview",
      wrappers: ["@/components/Auth"],
    },
    {
      path: "/media",
      name: "素材库",
      component: "./Media",
      wrappers: ["@/components/Auth"],
    },
    {
      path: "/articles",
      name: "文章管理",
      component: "./Articles",
      wrappers: ["@/components/Auth"],
    },
    {
      path: "/settings",
      name: "官网配置",
      component: "./Settings",
      wrappers: ["@/components/Auth"],
    },
    {
      path: "/products",
      name: "产品管理",
      component: "./Products",
      wrappers: ["@/components/Auth"],
    },
  ],
  proxy: {
    "/backend": {
      target: "http://127.0.0.1:8080",
      changeOrigin: false,
      pathRewrite: { "^/backend": "" },
    },
  },
});
