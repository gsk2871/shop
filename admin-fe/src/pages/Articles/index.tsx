import { useEffect, useState } from "react";
import {
	Button,
	Card,
	Form,
	Input,
	Modal,
	Popconfirm,
	Select,
	Space,
	Table,
	Tag,
	message,
} from "antd";
import { api, type Article } from "../../services/cms";
import MediaSelect from "../../components/MediaSelect";
export default function ArticlesPage() {
	const [rows, setRows] = useState<Article[]>([]);
	const [open, setOpen] = useState(false);
	const [busy, setBusy] = useState(false);
	const [editing, setEditing] = useState<Article>();
	const [form] = Form.useForm();
	const [msg, context] = message.useMessage();
	async function load() {
		try {
			setRows(await api<Article[]>("articles"));
		} catch (e) {
			msg.error((e as Error).message);
		}
	}
	useEffect(() => {
		void load();
	}, []);
	function edit(a?: Article) {
		setEditing(a);
		form.resetFields();
		form.setFieldsValue(
			a ?? {
				title: "",
				slug: "",
				summary: "",
				body: "",
				cover: "",
				status: "draft",
				seoTitle: "",
				seoDescription: "",
			},
		);
		setOpen(true);
	}
	return (
		<Card
			title="文章管理"
			extra={
				<Button type="primary" onClick={() => edit()}>
					新增文章
				</Button>
			}
		>
			{context}
			<Table
				rowKey="id"
				dataSource={rows}
				columns={[
					{ title: "标题", dataIndex: "title" },
					{ title: "Slug", dataIndex: "slug" },
					{
						title: "状态",
						render: (_, r) => (
							<Tag color={r.status === "published" ? "green" : "default"}>
								{r.status}
							</Tag>
						),
					},
					{
						title: "操作",
						render: (_, r) => (
							<Space>
								<Button onClick={() => edit(r)}>编辑</Button>
								<Popconfirm
									title="确定删除文章？"
									onConfirm={async () => {
										try {
											await api(`articles/${r.id}`, "DELETE");
											await load();
										} catch (e) {
											msg.error((e as Error).message);
										}
									}}
								>
									<Button danger>删除</Button>
								</Popconfirm>
							</Space>
						),
					},
				]}
			/>
			<Modal
				title={editing ? "编辑文章" : "新增文章"}
				open={open}
				width={800}
				onCancel={() => setOpen(false)}
				confirmLoading={busy}
				onOk={() => form.submit()}
				destroyOnHidden
			>
				<Form
					form={form}
					layout="vertical"
					onFinish={async (v) => {
						setBusy(true);
						try {
							await api("articles", editing ? "PUT" : "POST", {
								...v,
								id: editing?.id ?? 0,
								updatedAt: editing?.updatedAt ?? new Date().toISOString(),
							});
							setOpen(false);
							await load();
							msg.success("已保存");
						} catch (e) {
							msg.error((e as Error).message);
						} finally {
							setBusy(false);
						}
					}}
				>
					<Form.Item
						name="title"
						label="英文标题"
						rules={[{ required: true, max: 200 }]}
					>
						<Input />
					</Form.Item>
					<Form.Item
						name="slug"
						label="Slug（小写英文、数字、连字符）"
						rules={[
							{
								required: true,
								max: 120,
								pattern: /^[a-z0-9]+(?:-[a-z0-9]+)*$/,
							},
						]}
					>
						<Input />
					</Form.Item>
					<Form.Item name="summary" label="摘要">
						<Input.TextArea />
					</Form.Item>
					<Form.Item name="cover" label="封面">
						<MediaSelect />
					</Form.Item>
					<Form.Item name="body" label="正文（纯文本，空行分段）">
						<Input.TextArea rows={12} />
					</Form.Item>
					<Form.Item
						name="status"
						label="发布状态"
						rules={[{ required: true }]}
					>
						<Select
							options={[
								{ label: "草稿", value: "draft" },
								{ label: "直接发布", value: "published" },
								{ label: "下线", value: "offline" },
							]}
						/>
					</Form.Item>
					<Form.Item name="seoTitle" label="SEO 标题">
						<Input />
					</Form.Item>
					<Form.Item name="seoDescription" label="SEO 描述">
						<Input.TextArea />
					</Form.Item>
				</Form>
			</Modal>
		</Card>
	);
}
