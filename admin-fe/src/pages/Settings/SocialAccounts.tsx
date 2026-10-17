import { useState } from "react";
import { Button, Form, Image, Input, Modal, Popconfirm, Select, Space, Switch, Table } from "antd";
import type { SocialAccount } from "../../services/contracts";
import { assetURL } from "../../services/cms";
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

export default function SocialAccounts({ accounts, editable, busy, onSave }: {
  accounts: SocialAccount[];
  editable: boolean;
  busy: boolean;
  onSave: (accounts: SocialAccount[]) => Promise<boolean>;
}) {
  const [editing, setEditing] = useState<number | null>(null);
  const [form] = Form.useForm<SocialAccount>();
  const open = (index: number) => {
    form.resetFields();
    form.setFieldsValue(index < 0
      ? { name: "", platform: "facebook", link: "", iconMediaId: "", icon: "", enabled: true }
      : accounts[index]);
    setEditing(index);
  };
  return (
    <>
      <p>已开启的账号显示在官网首页页脚，按列表顺序排列。支持同一平台的多个账号，开关切换后立即保存。</p>
      {editable && <Button onClick={() => open(-1)} disabled={busy || accounts.length >= 50}>新增社交账号</Button>}
      <Table
        style={{ marginTop: 16 }}
        dataSource={accounts.map((account, index) => ({ ...account, index }))}
        rowKey="index"
        pagination={false}
        scroll={{ x: 800 }}
        columns={[
          { title: "名称", dataIndex: "name" },
          { title: "平台", dataIndex: "platform", render: (value) => platforms.find((p) => p.value === value)?.label || value },
          { title: "链接", dataIndex: "link", ellipsis: true },
          { title: "图标", dataIndex: "icon", render: (value, row) => value && <Image width={32} height={32} style={{ objectFit: "contain" }} src={assetURL(value)} alt={row.name} /> },
          { title: "是否开启", dataIndex: "enabled", render: (value, row) => (
            <Switch checked={value} disabled={!editable || busy} onChange={(enabled) => void onSave(accounts.map((account, index) => index === row.index ? { ...account, enabled } : account))} />
          ) },
          { title: "操作", render: (_, row) => editable && (
            <Space>
              <Button disabled={busy} onClick={() => open(row.index)}>编辑</Button>
              <Popconfirm title="删除此社交账号？" disabled={busy} onConfirm={() => onSave(accounts.filter((_, index) => index !== row.index)).then(() => undefined)}>
                <Button danger disabled={busy}>删除</Button>
              </Popconfirm>
            </Space>
          ) },
        ]}
      />
      <Modal title={editing === -1 ? "新增社交账号" : "编辑社交账号"} open={editing !== null}
        onCancel={() => { if (!busy) setEditing(null); }}
        onOk={() => form.submit()} confirmLoading={busy} cancelButtonProps={{ disabled: busy }}>
        <Form form={form} layout="vertical" disabled={!editable || busy} onFinish={async (values) => {
          if (editing === null) return;
          const next = editing < 0 ? [...accounts, values] : accounts.map((account, index) => index === editing ? values : account);
          if (await onSave(next)) setEditing(null);
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
          <Form.Item name="enabled" label="是否开启" valuePropName="checked"><Switch /></Form.Item>
        </Form>
      </Modal>
    </>
  );
}
