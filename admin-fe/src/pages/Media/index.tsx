import { useEffect, useState } from "react";
import { Button, Card, Image, Space, Table, Upload, message } from "antd";
import { api, assetURL, type Media } from "../../services/cms";
export default function MediaPage() {
	const [rows, setRows] = useState<Media[]>([]);
	const [busy, setBusy] = useState(false);
	const [msg, context] = message.useMessage();
	async function load() {
		try {
			setRows(await api<Media[]>("media"));
		} catch (e) {
			msg.error((e as Error).message);
		}
	}
	useEffect(() => {
		void load();
	}, []);
	return (
		<Card
			title="素材库"
			extra={
				<Space>
					<Button onClick={load}>刷新</Button>
					<Upload
						accept="image/jpeg,image/png,image/webp,image/gif,application/pdf,video/mp4,video/webm"
						showUploadList={false}
						customRequest={async ({ file, onSuccess, onError }) => {
							setBusy(true);
							try {
								const form = new FormData();
								form.append("file", file as File);
								const r = await fetch("/backend/admin/api/v1/media", {
									method: "POST",
									headers: { "X-Requested-With": "peptide-admin" },
									body: form,
								});
								const data = await r.json();
								if (!r.ok) throw new Error(data.error);
								onSuccess?.(data);
								msg.success("上传成功");
								await load();
							} catch (e) {
								onError?.(e as Error);
								msg.error((e as Error).message);
							} finally {
								setBusy(false);
							}
						}}
					>
						<Button type="primary" loading={busy}>
							上传素材
						</Button>
					</Upload>
				</Space>
			}
		>
			{context}
			<p>
				支持 JPG、PNG、WebP、GIF、PDF、MP4、WebM，单文件最多
				20MB。仅上传与展示，无转码任务。
			</p>
			<Table
				rowKey="id"
				dataSource={rows}
				columns={[
					{
						title: "预览",
						render: (_, r) =>
							r.mime.startsWith("image/") ? (
								<Image width={72} src={assetURL(r.url)} />
							) : (
								<a href={assetURL(r.url)} target="_blank" rel="noreferrer">
									打开素材
								</a>
							),
					},
					{ title: "名称", dataIndex: "name" },
					{ title: "类型", dataIndex: "mime" },
					{
						title: "大小",
						render: (_, r) => `${(r.size / 1024).toFixed(1)} KB`,
					},
					{
						title: "上传时间",
						render: (_, r) => new Date(r.createdAt).toLocaleString(),
					},
					{
						title: "地址",
						render: (_, r) => (
							<Button
								onClick={async () => {
									try {
										await navigator.clipboard.writeText(r.url);
										msg.success("已复制");
									} catch {
										msg.error("复制失败，请手动复制：" + r.url);
									}
								}}
							>
								复制引用地址
							</Button>
						),
					},
				]}
			/>
		</Card>
	);
}
