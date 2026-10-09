import { defineConfig } from "@umijs/max";

export default defineConfig({
	title: "多肽站管理后台",
	hash: true,
	esbuildMinifyIIFE: true,
	antd: {},
	layout: { title: "多肽站管理后台" },
	request: {},
	routes: [
		{ path: "/login", component: "./Login", layout: false },
		{ path: "/", redirect: "/overview" },
		{
			path: "/overview",
			name: "工作台",
			component: "./Overview",
			wrappers: ["@/components/Auth"],
		},
		{
			path: "/system",
			name: "系统管理",
			component: "./System",
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
		{ path: "/products", name: "产品管理", component: "./Products", wrappers: ["@/components/Auth"] },
		...[
			["categories", "分类管理"],
			["reports", "检测报告"],
			["inquiries", "询盘管理"],
			["contact-routing", "WhatsApp 分流"],
		].map(([path, name]) => ({
			path: `/${path}`,
			name,
			component: "./Placeholder",
			wrappers: ["@/components/Auth"],
		})),
	],
	proxy: {
		"/backend": {
			target: "http://127.0.0.1:8080",
			changeOrigin: true,
			pathRewrite: { "^/backend": "" },
		},
	},
});
