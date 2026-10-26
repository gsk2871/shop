import { useEffect, useState } from "react";
import { useModel } from "@umijs/max";
import { Alert, Button, Card, Form, Input, Space, Switch, message } from "antd";
import { api } from "../../services/cms";
import { allows } from "../../services/auth";
import type { LeadPopupConfig } from "../../services/contracts";

const fields = [
  ["title", "主标题", 200], ["description", "说明文案", 2000],
  ["placeholder", "输入框提示", 200], ["submitText", "提交按钮文案", 200],
  ["dismissText", "拒绝按钮文案", 200], ["notice", "底部说明", 2000],
  ["successTitle", "提交成功标题", 200], ["successDescription", "提交成功说明", 2000],
] as const;
export default function LeadPopupPage() {
  const { initialState } = useModel("@@initialState");
  const canEdit = allows(initialState?.identity, "lead_popup.update");
  const [form] = Form.useForm<LeadPopupConfig>();
  const [config, setConfig] = useState<LeadPopupConfig>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [msg, context] = message.useMessage();
  async function load() {
    try {
      const saved = await api<LeadPopupConfig>("lead-popup");
      setConfig(saved); form.setFieldsValue(saved); setError("");
    } catch (e) { setError((e as Error).message); }
  }
  useEffect(() => { void load(); }, []);
  return <Card title="留资弹窗配置" extra={<Button disabled={busy} onClick={() => void load()}>刷新</Button>}>
    {context}
    {error && <Alert type="error" title={error} />}
    {config?.version === 0 && <Alert type="warning" title="请先执行后端种子初始化，再刷新此页面。" />}
    <p>每次访问立即展示，同一次访问关闭或提交后不再弹出。以下内容以英文展示给访客。</p>
    <Form form={form} layout="vertical" disabled={!canEdit || busy || !config?.version} style={{ maxWidth: 800 }} onFinish={async (values) => {
      setBusy(true);
      try {
        const saved = await api<LeadPopupConfig>("lead-popup", "PUT", { ...values, expectedVersion: config?.version });
        setConfig(saved); form.setFieldsValue(saved); msg.success("已保存");
      } catch (e) { msg.error((e as Error).message); }
      finally { setBusy(false); }
    }}>
      <Form.Item name="enabled" label="弹窗开关" valuePropName="checked"><Switch /></Form.Item>
      {fields.map(([key, label, max]) => <Form.Item key={key} name={key} label={label} rules={[{ required: true, whitespace: true, max }]}>
        {max > 200 ? <Input.TextArea rows={3} maxLength={max} showCount /> : <Input maxLength={max} />}
      </Form.Item>)}
      {canEdit && <Space><Button type="primary" htmlType="submit" loading={busy}>保存配置</Button></Space>}
    </Form>
  </Card>;
}
