import { Button, Card, Form, Input, message } from "antd";
import { history } from "@umijs/max";
import { api } from "../../services/cms";
export default function Login() {
	const [msg, context] = message.useMessage();
	return (
		<div
			style={{
				minHeight: "100vh",
				display: "grid",
				placeItems: "center",
				background: "#f2f5f3",
			}}
		>
			{context}
			<Card title="管理后台登录" style={{ width: 380, maxWidth: "95vw" }}>
				<Form
					onFinish={async (values) => {
						try {
							await api("login", "POST", values);
							history.replace("/overview");
						} catch (e) {
							msg.error((e as Error).message);
						}
					}}
					layout="vertical"
				>
					<Form.Item name="username" label="账号" rules={[{ required: true }]}>
						<Input autoComplete="username" />
					</Form.Item>
					<Form.Item name="password" label="密码" rules={[{ required: true }]}>
						<Input.Password autoComplete="current-password" />
					</Form.Item>
					<Button type="primary" htmlType="submit" block>
						登录
					</Button>
				</Form>
			</Card>
		</div>
	);
}
