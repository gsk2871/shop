import { useLocation } from "@umijs/max";
import { Alert, Card } from "antd";

const modules: Record<string, string> = {
	"/products": "产品参数、英文内容、搜索、上下架",
	"/categories": "产品分类与排序",
	"/reports": "产品名称与检测报告图片",
	"/articles": "文章直接发布、SEO 字段",
	"/inquiries": "询盘列表、详情与处理状态；表单字段待确认",
	"/contact-routing": "国家与 WhatsApp 号码映射、默认号码",
	"/settings": "品牌、公司信息、社媒链接、Cookie 提示、科研声明",
	"/account": "单管理员账户、登录与安全设置",
};

export default function Placeholder() {
	const { pathname } = useLocation();
	return (
		<Card title={modules[pathname] ?? "模块"}>
			<Alert
				type="info"
				message="此模块尚未实现"
				description="框架仅提供菜单和页面入口，未连接业务 API，也不会保存数据。"
			/>
		</Card>
	);
}
