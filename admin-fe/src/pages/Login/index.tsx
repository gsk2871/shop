import { useState } from "react";
import { Alert, Button, Card, Form, Input, message } from "antd";
import { history, useModel } from "@umijs/max";
import { api, APIException } from "../../services/cms";
import { me } from "../../services/auth";
interface LoginResult {
  step: "totp" | "session";
  ticket?: string;
  scope?: string;
  expiresIn: number;
}
export default function Login() {
  const [ticket, setTicket] = useState("");
  const [expires, setExpires] = useState(0);
  const [busy, setBusy] = useState(false);
  const [msg, context] = message.useMessage();
  const { setInitialState } = useModel("@@initialState");
  async function finish(result: LoginResult) {
    if (result.step === "totp") {
      setTicket(result.ticket || "");
      setExpires(Date.now() + result.expiresIn * 1000);
      return;
    }
    const identity = await me();
    await setInitialState({ identity });
    history.replace(
      identity.scope === "full" ? "/overview" : "/system/security",
    );
  }
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
        {ticket ? (
          <>
            <Alert type="info" title="输入验证器中的 6 位验证码" />
            <Form
              layout="vertical"
              onFinish={async (values) => {
                setBusy(true);
                try {
                  if (Date.now() > expires) {
                    setTicket("");
                    throw new Error("登录票据已过期，请重新输入密码");
                  }
                  await finish(
                    await api<LoginResult>("login/totp", "POST", {
                      ticket,
                      code: values.code,
                    }),
                  );
                } catch (e) {
                  msg.error((e as Error).message);
                  if (
                    e instanceof APIException &&
                    e.code === "MFA_TICKET_INVALID"
                  )
                    setTicket("");
                } finally {
                  setBusy(false);
                }
              }}
            >
              <Form.Item
                name="code"
                label="验证码"
                rules={[{ required: true, pattern: /^\d{6}$/ }]}
              >
                <Input
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  maxLength={6}
                />
              </Form.Item>
              <Button loading={busy} htmlType="submit" type="primary" block>
                验证并登录
              </Button>
              <Button block onClick={() => setTicket("")}>
                返回密码登录
              </Button>
            </Form>
          </>
        ) : (
          <Form
            layout="vertical"
            onFinish={async (values) => {
              setBusy(true);
              try {
                await finish(await api<LoginResult>("login", "POST", values));
              } catch (e) {
                msg.error((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <Form.Item
              name="username"
              label="账号"
              rules={[{ required: true }]}
            >
              <Input autoComplete="username" />
            </Form.Item>
            <Form.Item
              name="password"
              label="密码"
              rules={[{ required: true }]}
            >
              <Input.Password autoComplete="current-password" />
            </Form.Item>
            <Button type="primary" htmlType="submit" loading={busy} block>
              登录
            </Button>
          </Form>
        )}
      </Card>
    </div>
  );
}
