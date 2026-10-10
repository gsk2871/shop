import { useEffect, useState } from "react";
import { useModel } from "@umijs/max";
import { Button, Card, Form, Input, Space, message } from "antd";
import { api, type Settings } from "../../services/cms";
import { defaultTheme } from "../../services/contracts";
import { allows } from "../../services/auth";
import MediaSelect from "../../components/MediaSelect";
export default function SettingsPage() {
  const { initialState } = useModel("@@initialState");
  const editable = allows(initialState?.identity, "site.update");
  const [form] = Form.useForm<Settings>();
  const [version, setVersion] = useState(1);
  const [busy, setBusy] = useState(false);
  const [msg, context] = message.useMessage();
  const theme = Form.useWatch("theme", form) || defaultTheme;
  useEffect(() => {
    api<Settings>("settings")
      .then((s) => {
        setVersion(s.version);
        form.setFieldsValue(s);
      })
      .catch((e) => msg.error(e.message));
  }, []);
  return (
    <Card title="品牌与主题">
      {context}
      <Form
        form={form}
        layout="vertical"
        disabled={!editable}
        onFinish={async (values) => {
          setBusy(true);
          try {
            const saved = await api<Settings>("settings", "PUT", {
              ...values,
              expectedVersion: version,
            });
            setVersion(saved.version);
            msg.success("已保存，官网下次请求生效");
          } catch (e) {
            msg.error((e as Error).message);
          } finally {
            setBusy(false);
          }
        }}
      >
        <Form.Item
          name="brand"
          label="英文品牌名称"
          rules={[{ required: true, max: 100 }]}
        >
          <Input />
        </Form.Item>
        <Form.Item name="logoMediaId" label="Logo 图片">
          <MediaSelect />
        </Form.Item>
        <Space wrap>
          {Object.entries({
            primary: "主色",
            background: "背景",
            surface: "内容底色",
            text: "正文",
            muted: "辅助文字",
            border: "边框",
            accent: "强调色",
            onPrimary: "主色上文字",
          }).map(([key, label]) => (
            <Form.Item
              key={key}
              name={["theme", key]}
              label={label}
              rules={[{ required: true, pattern: /^#[0-9a-fA-F]{6}$/ }]}
            >
              <Input type="color" />
            </Form.Item>
          ))}
        </Space>
        <div
          style={{
            background: theme.background,
            color: theme.text,
            padding: 24,
            border: `1px solid ${theme.border}`,
            marginBottom: 24,
          }}
        >
          <h3>Research Supply</h3>
          <p style={{ color: theme.muted }}>主题预览 · 辅助文字</p>
          <div style={{ background: theme.surface, padding: 16 }}>
            内容区域 <span style={{ color: theme.accent }}>强调内容</span>
          </div>
          <button
            type="button"
            style={{
              background: theme.primary,
              color: theme.onPrimary,
              padding: 12,
              border: 0,
            }}
          >
            Contact us
          </button>
        </div>
        <Button
          onClick={() => form.setFieldValue("theme", { ...defaultTheme })}
        >
          恢复默认主题
        </Button>
        <Form.Item name="description" label="英文公司介绍">
          <Input.TextArea maxLength={5000} />
        </Form.Item>
        <Form.Item name="email" label="联系邮箱" rules={[{ type: "email" }]}>
          <Input />
        </Form.Item>
        <Form.Item
          name="whatsapp"
          label="WhatsApp 国际号码（仅数字）"
          rules={[{ pattern: /^$|^[0-9]{7,15}$/ }]}
        >
          <Input />
        </Form.Item>
        <Form.Item name="disclaimer" label="英文科研声明">
          <Input.TextArea maxLength={500} />
        </Form.Item>
        {editable && (
          <Button type="primary" htmlType="submit" loading={busy}>
            保存品牌与主题
          </Button>
        )}
      </Form>
    </Card>
  );
}
