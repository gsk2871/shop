import { useState } from "react";
import { history, useModel } from "@umijs/max";
import { Alert, Button, Card, Form, Input, QRCode, Space, message } from "antd";
import { api } from "../../services/cms";
export default function Security() {
  const { initialState } = useModel("@@initialState");
  const identity = initialState?.identity;
  const [binding, setBinding] = useState<{ secret: string; uri: string }>();
  const [busy, setBusy] = useState(false);
  const [msg, context] = message.useMessage();
  async function invoke(action: () => Promise<unknown>) {
    setBusy(true);
    try {
      await action();
    } catch (e) {
      msg.error((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  const relogin = () => {
    msg.success("操作成功，请重新登录");
    history.replace("/login");
  };
  return (
    <Space orientation="vertical" style={{ width: "100%" }}>
      {context}
      <Card
        title="个人安全中心"
        extra={
          <Button
            onClick={() =>
              void invoke(async () => {
                await api("logout", "POST");
                history.replace("/login");
              })
            }
          >
            退出登录
          </Button>
        }
      >
        {identity?.scope !== "full" && (
          <Alert
            type="info"
            title={
              identity?.user.mustChangePwd
                ? "请先修改初始密码，再重新登录并绑定验证器"
                : "请绑定验证器，完成后重新登录"
            }
          />
        )}
        {(identity?.user.mustChangePwd || identity?.scope === "full") && (
          <Form
            layout="vertical"
            onFinish={(values) =>
              void invoke(async () => {
                await api("password", "POST", values);
                relogin();
              })
            }
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
              rules={[
                { required: true },
                {
                  validator: (_, value: string) =>
                    !value ||
                    (new TextEncoder().encode(value).length >= 12 &&
                      new TextEncoder().encode(value).length <= 72)
                      ? Promise.resolve()
                      : Promise.reject(new Error("密码需为 12–72 字节")),
                },
              ]}
            >
              <Input.Password autoComplete="new-password" />
            </Form.Item>
            <Button type="primary" htmlType="submit" loading={busy}>
              修改密码
            </Button>
          </Form>
        )}
        {!identity?.user.mustChangePwd && identity?.scope === "onboarding" && (
          <>
            <Button
              loading={busy}
              onClick={() =>
                void invoke(async () =>
                  setBinding(await api("totp/setup", "POST")),
                )
              }
            >
              生成验证器绑定信息
            </Button>
            {binding && (
              <>
                <p>使用验证器扫描二维码，或手动录入密钥。</p>
                <QRCode value={binding.uri} />
                <Input readOnly value={binding.secret} />
                <Form
                  onFinish={(values) =>
                    void invoke(async () => {
                      await api("totp/activate", "POST", values);
                      setBinding(undefined);
                      relogin();
                    })
                  }
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
                  <Button type="primary" htmlType="submit" loading={busy}>
                    确认绑定
                  </Button>
                </Form>
              </>
            )}
          </>
        )}
      </Card>
      {identity?.scope === "full" && (
        <Card title="敏感操作身份验证">
          <p>
            重置密码或验证器前，先在此验证身份。验证有效期为 5
            分钟，同一验证码不能重复使用。
          </p>
          <Form
            layout="vertical"
            onFinish={(values) =>
              void invoke(async () => {
                await api("me/reauth", "POST", values);
                msg.success("身份验证成功");
              })
            }
            style={{ maxWidth: 420 }}
          >
            <Form.Item
              name="password"
              label="当前密码"
              rules={[{ required: true }]}
            >
              <Input.Password autoComplete="current-password" />
            </Form.Item>
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
            <Button htmlType="submit" loading={busy}>
              再次验证
            </Button>
          </Form>
        </Card>
      )}
    </Space>
  );
}
