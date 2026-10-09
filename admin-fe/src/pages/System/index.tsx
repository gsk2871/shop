import { useEffect, useState } from "react";
import { Button, Card, Form, Input, Space, Table, message } from "antd";
import { history } from "@umijs/max";
import { api } from "../../services/cms";
interface Audit {
	id: number;
	action: string;
	createdAt: string;
}
export default function SystemPage() {
	const [user, setUser] = useState("");
	const [logs, setLogs] = useState<Audit[]>([]);
	const [msg, context] = message.useMessage();
	useEffect(() => {
		api<{ username: string }>("me")
			.then((v) => setUser(v.username))
			.catch((e: Error) => msg.error(e.message));
		api<Audit[]>("audit")
			.then(setLogs)
			.catch((e: Error) => msg.error(e.message));
	}, []);
	return (
		<Space direction="vertical" style={{ width: "100%" }}>
			{context}
			<Card
				title={`系统管理 · ${user}`}
				extra={
					<Button
						onClick={async () => {
							try {
								await api("logout", "POST");
								history.replace("/login");
							} catch (e) {
								msg.error((e as Error).message);
							}
						}}
					>
						退出登录
					</Button>
				}
			>
				<p>单管理员账号。修改密码后，所有登录会话失效。</p>
				<Form
					layout="vertical"
					onFinish={async (v) => {
						try {
							await api("password", "POST", v);
							msg.success("密码已修改，请重新登录");
							history.replace("/login");
						} catch (e) {
							msg.error((e as Error).message);
						}
					}}
					style={{ maxWidth: 420 }}
				>
					<Form.Item
						name="current"
						label="当前密码"
						rules={[{ required: true }]}
					>
						<Input.Password autoComplete="current-password" />
					</Form.Item>
					<Form.Item
						name="password"
						label="新密码（12–72 字节）"
						rules={[{ required: true, min: 12, max: 72 }]}
					>
						<Input.Password autoComplete="new-password" />
					</Form.Item>
					<Button type="primary" htmlType="submit">
						修改密码
					</Button>
				</Form>
			</Card>
			<Card title="最近操作日志">
				<Table
					rowKey="id"
					dataSource={logs}
					columns={[
						{ title: "操作", dataIndex: "action" },
						{
							title: "时间",
							render: (_, r) => new Date(r.createdAt).toLocaleString(),
						},
					]}
				/>
			</Card>
		</Space>
	);
}
