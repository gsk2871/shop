import { defineConfig } from "@umijs/max";

export default defineConfig({
  title: "管理后台",
  hash: true,
  esbuildMinifyIIFE: true,
  antd: {
    configProvider: {
      theme: {
        token: {
          colorPrimary: "#1677ff",
          borderRadius: 8,
          colorBgLayout: "#f5f7fa",
        },
      },
    },
  },
  layout: { title: "管理后台" },
  request: {},
  initialState: {},
  model: {},
  routes: [
    { path: "/lead-popup", name: "留资弹窗配置", component: "./LeadPopup", wrappers: ["@/components/Auth"] },
    { path: "/leads", name: "留资记录", component: "./Leads", wrappers: ["@/components/Auth"] },
    {
      path: "/faqs",
      name: "问答配置",
      component: "./FAQs",
      wrappers: ["@/components/Auth"],
    },
    {
      path: "/categories",
      name: "商品分类",
      component: "./Categories",
      wrappers: ["@/components/Auth"],
    },
    {
      path: "/whatsapp",
      name: "客服配置",
      component: "./WhatsApp",
      wrappers: ["@/components/Auth"],
    },
    {
      path: "/banners",
      name: "Banner管理",
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
      name: "角色权限管理",
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
      name: "个人中心",
      component: "./Security",
      wrappers: ["@/components/Auth"],
    },
    {
      path: "/system/config",
      name: "运行配置状态",
      component: "./Config",
      wrappers: ["@/components/Auth"],
    },
    { path: "/login", component: "./Login", layout: false },
    { path: "/", component: "./Home" },
    { path: "/overview", redirect: "/" },
    {
      path: "/media",
      name: "媒体库",
      component: "./Media",
      wrappers: ["@/components/Auth"],
    },
    {
      path: "/articles",
      name: "新闻管理",
      component: "./Articles",
      wrappers: ["@/components/Auth"],
    },
    {
      path: "/settings",
      name: "品牌与主题",
      component: "./Settings",
      wrappers: ["@/components/Auth"],
    },
    {
      path: "/products",
      name: "商品管理",
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
