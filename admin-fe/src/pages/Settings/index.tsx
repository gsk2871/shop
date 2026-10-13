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
  const [savedSettings, setSavedSettings] = useState<Settings>();
  const [busy, setBusy] = useState(false);
  const [msg, context] = message.useMessage();
  const brand = Form.useWatch("brand", form);
  const theme = Form.useWatch("theme", form) || defaultTheme;
  useEffect(() => {
    api<Settings>("settings")
      .then((s) => {
        setVersion(s.version);
        setSavedSettings(s);
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
              ...savedSettings,
              ...values,
              expectedVersion: version,
            });
            setVersion(saved.version);
            setSavedSettings(saved);
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
        <Form.Item name="brandMark" label="品牌文字标记（未设置 Logo 时显示，如 R·）" rules={[{ max: 200 }]}>
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
          <h3>{brand}</h3>
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
        <p>Home 主视觉在「Banner管理」的 home_hero 配置。Products 数据来自「商品管理」；News 展示「新闻管理」中已发布的文章，页头使用 research_header 投放位。页面正文为纯文本，保留换行；留空不展示。</p>
        {([
          ["homeTitle", "Home 介绍标题", 200],
          ["homeContent", "Home 介绍正文", 20000],
          ["aboutTitle", "About 标题", 200],
          ["aboutContent", "About 正文", 20000],
          ["productsTitle", "Products 标题", 200],
          ["newsTitle", "News 标题", 200],
          ["privacyTitle", "隐私页面标题", 200],
          ["privacyContent", "隐私页面正文", 20000],
          ["cookieTitle", "Cookie 提示标题", 200],
          ["cookieContent", "Cookie 提示正文", 20000],
        ] as const).map(([name, label, max]) => (
          <Form.Item key={name} name={name} label={label} rules={[{ max }]}>
            {max > 200 ? <Input.TextArea rows={6} maxLength={max} /> : <Input maxLength={max} />}
          </Form.Item>
        ))}
        {editable && (
          <Button type="primary" htmlType="submit" loading={busy}>
            保存品牌与主题
          </Button>
        )}
      </Form>
    </Card>
  );
}
