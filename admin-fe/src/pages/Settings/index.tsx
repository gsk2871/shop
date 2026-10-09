import { useEffect, useState } from "react";
import { Button, Card, Form, Input, Space, message } from "antd";
import { api, type Settings } from "../../services/cms";
import MediaSelect from "../../components/MediaSelect";
export default function SettingsPage() {
	const [form] = Form.useForm<Settings>();
	const [busy, setBusy] = useState(false);
	const [msg, context] = message.useMessage();
	useEffect(() => {
		api<Settings>("settings")
			.then((v) => form.setFieldsValue(v))
			.catch((e: Error) => msg.error(e.message));
	}, []);
	return (
		<Card title="官网配置">
			{context}
			<Form
				form={form}
				layout="vertical"
				onFinish={async (v) => {
					setBusy(true);
					try {
						await api("settings", "PUT", v);
						msg.success("保存成功，官网下次请求会加载新配置");
					} catch (e) {
						msg.error((e as Error).message);
					} finally {
						setBusy(false);
					}
				}}
			>
				<Form.Item
					name="brand"
					label="品牌名称"
					rules={[{ required: true, max: 100 }]}
				>
					<Input />
				</Form.Item>
				<Form.Item name="logo" label="Logo">
					<MediaSelect />
				</Form.Item>
				<Space align="start">
					<Form.Item
						name="primaryColor"
						label="主色（#RRGGBB）"
						rules={[{ required: true, pattern: /^#[0-9a-fA-F]{6}$/ }]}
					>
						<Input type="color" />
					</Form.Item>
					<Form.Item
						name="backgroundColor"
						label="背景色"
						rules={[{ required: true, pattern: /^#[0-9a-fA-F]{6}$/ }]}
					>
						<Input type="color" />
					</Form.Item>
				</Space>
				<Form.Item name="description" label="公司介绍">
					<Input.TextArea rows={3} />
				</Form.Item>
				<Form.Item name="email" label="联系邮箱" rules={[{ type: "email" }]}>
					<Input />
				</Form.Item>
				<Form.Item
					name="whatsapp"
					label="默认 WhatsApp 号码（国际号码，仅数字）"
					rules={[{ pattern: /^$|^[0-9]{7,15}$/ }]}
				>
					<Input />
				</Form.Item>
				<Form.Item name="disclaimer" label="科研声明" rules={[{ max: 500 }]}>
					<Input.TextArea />
				</Form.Item>
				<Form.List name="banners">
					{(fields, { add, remove }) => (
						<>
							<h3>首页 Banner（最多 3 张）</h3>
							{fields.map((field) => (
								<Card
									key={field.key}
									size="small"
									style={{ marginBottom: 16 }}
									extra={
										<Button danger onClick={() => remove(field.name)}>
											删除
										</Button>
									}
								>
									<Form.Item
										name={[field.name, "title"]}
										label="标题"
										rules={[{ required: true, max: 200 }]}
									>
										<Input />
									</Form.Item>
									<Form.Item name={[field.name, "subtitle"]} label="副标题">
										<Input.TextArea />
									</Form.Item>
									<Form.Item name={[field.name, "image"]} label="图片">
										<MediaSelect />
									</Form.Item>
									<Form.Item
										name={[field.name, "link"]}
										label="按钮链接（站内路径或 HTTPS）"
									>
										<Input />
									</Form.Item>
								</Card>
							))}
							<Button
								disabled={fields.length >= 3}
								onClick={() =>
									add({ title: "", subtitle: "", image: "", link: "/contact" })
								}
							>
								添加 Banner
							</Button>
						</>
					)}
				</Form.List>
				<Button
					type="primary"
					htmlType="submit"
					loading={busy}
					style={{ marginTop: 24 }}
				>
					保存官网配置
				</Button>
			</Form>
		</Card>
	);
}
