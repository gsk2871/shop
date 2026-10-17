import { useEffect, useRef, useState } from "react";
import { Button, Form, Image, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Table, message } from "antd";
import type { SocialAccount } from "../../services/contracts";
import { api, assetURL } from "../../services/cms";
import MediaSelect from "../../components/MediaSelect";

const platforms = [
  { value: "facebook", label: "Facebook" },
  { value: "tiktok", label: "TikTok" },
  { value: "youtube", label: "YouTube" },
  { value: "x", label: "X" },
  { value: "instagram", label: "Instagram" },
  { value: "linkedin", label: "LinkedIn" },
  { value: "other", label: "其他" },
];

export default function SocialAccounts({ editable }: { editable: boolean }) {
  const [accounts, setAccounts] = useState<SocialAccount[]>([]);
  const [busy, setBusy] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [editing, setEditing] = useState<SocialAccount | null | undefined>();
  const [msg, context] = message.useMessage();
  const pending = useRef(false);
  const refresh = async () => {
    if (pending.current) return;
    pending.current = true;
    setBusy(true);
    setLoaded(false);
    try {
      const result = await api<{ items: SocialAccount[] }>("social-accounts");
      setAccounts(result.items);
      setLoaded(true);
    } catch (e) { msg.error((e as Error).message); }
    finally { pending.current = false; setBusy(false); }
  };
  useEffect(() => { void refresh(); }, []);
  const save = async (path: string, method: string, body: unknown, removeID?: string) => {
    if (pending.current || !editable || !loaded) return false;
    pending.current = true;
    setBusy(true);
    try {
      const saved = await api<SocialAccount>(path, method, body);
      setAccounts((previous) => removeID
        ? previous.filter((account) => account.id !== removeID)
        : [...previous.filter((account) => account.id !== saved.id), saved].sort((a, b) => a.sort - b.sort || (BigInt(a.id) < BigInt(b.id) ? -1 : 1)));
      msg.success("社交账号已保存，刷新官网首页生效");
      return true;
    } catch (e) { msg.error((e as Error).message); return false; }
    finally { pending.current = false; setBusy(false); }
  };
  const [form] = Form.useForm<SocialAccount>();
  const open = (account: SocialAccount | null) => {
    form.resetFields();
    form.setFieldsValue(account || { name: "", platform: "facebook", link: "", iconMediaId: "", icon: "", enabled: true, sort: 0 });
    setEditing(account);
  };
  return (
    <>
      {context}
      <p>已开启的账号显示在官网首页页脚，按排序值从小到大排列，同序按创建顺序排列。支持同一平台的多个账号，开关切换后立即保存。</p>
      <Space>
        {editable && <Button onClick={() => open(null)} disabled={busy || !loaded || accounts.length >= 50}>新增社交账号</Button>}
        <Button onClick={() => void refresh()} disabled={busy}>刷新列表</Button>
      </Space>
      <Table
        style={{ marginTop: 16 }}
        dataSource={accounts}
        loading={busy}
        rowKey="id"
        pagination={false}
        scroll={{ x: 800 }}
        columns={[
          { title: "名称", dataIndex: "name" },
          { title: "平台", dataIndex: "platform", render: (value) => platforms.find((p) => p.value === value)?.label || value },
          { title: "链接", dataIndex: "link", ellipsis: true },
          { title: "图标", dataIndex: "icon", render: (value, row) => value && <Image width={32} height={32} style={{ objectFit: "contain" }} src={assetURL(value)} alt={row.name} /> },
          { title: "是否开启", dataIndex: "enabled", render: (value, row) => (
            <Switch checked={value} disabled={!editable || busy || !loaded} onChange={(enabled) => void save(`social-accounts/${row.id}/enabled`, "PATCH", { enabled, expectedVersion: row.version })} />
          ) },
          { title: "排序", dataIndex: "sort" },
          { title: "操作", render: (_, row) => editable && (
            <Space>
              <Button disabled={busy || !loaded} onClick={() => open(row)}>编辑</Button>
              <Popconfirm title="删除此社交账号？" disabled={busy || !loaded} onConfirm={() => save(`social-accounts/${row.id}`, "DELETE", { expectedVersion: row.version }, row.id).then(() => undefined)}>
                <Button danger disabled={busy || !loaded}>删除</Button>
              </Popconfirm>
            </Space>
          ) },
        ]}
      />
      <Modal title={editing === null ? "新增社交账号" : "编辑社交账号"} open={editing !== undefined}
        onCancel={() => { if (!busy) setEditing(undefined); }}
        onOk={() => form.submit()} confirmLoading={busy} cancelButtonProps={{ disabled: busy }}>
        <Form form={form} layout="vertical" disabled={!editable || busy || !loaded} onFinish={async (values) => {
          if (editing === undefined) return;
          if (await save(editing ? `social-accounts/${editing.id}` : "social-accounts", editing ? "PUT" : "POST", { ...values, expectedVersion: editing?.version || 0 })) setEditing(undefined);
        }}>
          <Form.Item name="name" label="名称" rules={[{ required: true, whitespace: true, max: 100 }]}><Input maxLength={100} /></Form.Item>
          <Form.Item name="platform" label="平台" rules={[{ required: true }]}><Select options={platforms} /></Form.Item>
          <Form.Item name="link" label="账号链接" rules={[{ required: true, max: 2000 }, { validator: (_, value) => {
            try {
              const url = new URL(value);
              if (url.protocol === "https:" && url.hostname && !url.username && !url.password && !/[\s\\]/.test(value)) return Promise.resolve();
            } catch {}
            return Promise.reject(new Error("请输入完整的 HTTPS 账号链接"));
          } }]}><Input maxLength={2000} placeholder="https://…" /></Form.Item>
          <Form.Item name="iconMediaId" label="图标" rules={[{ required: true, message: "请选择图标" }]}><MediaSelect /></Form.Item>
          <Form.Item name="sort" label="排序" rules={[{ required: true }]}><InputNumber precision={0} min={-2147483648} max={2147483647} /></Form.Item>
          <Form.Item name="enabled" label="是否开启" valuePropName="checked"><Switch /></Form.Item>
        </Form>
      </Modal>
    </>
  );
}
