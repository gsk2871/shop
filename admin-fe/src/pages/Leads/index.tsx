import { useEffect, useRef, useState } from "react";
import { useModel } from "@umijs/max";
import { Button, Card, DatePicker, Form, Input, Modal, Select, Space, Table, Tag, Typography, message } from "antd";
import type { DatePickerProps } from "antd";
import { api } from "../../services/cms";
import { allows } from "../../services/auth";
import type { ContactLead, Page } from "../../services/contracts";

const statuses = { pending: "待跟进", completed: "已完成", invalid: "无效" };
type PickerDate = NonNullable<DatePickerProps["minDate"]>;
type Filters = { contact?: string; start?: PickerDate; end?: PickerDate };
const displayTime = (value: string) => new Date(value).toLocaleString("zh-CN", { timeZone: "Asia/Shanghai", hour12: false });
export default function LeadsPage() {
  const { initialState } = useModel("@@initialState");
  const canEdit = allows(initialState?.identity, "lead.update");
  const [filters] = Form.useForm<Filters>();
  const [form] = Form.useForm();
  const [data, setData] = useState<Page<ContactLead>>({ items: [], page: 1, pageSize: 20, total: 0 });
  const query = useRef<Filters>({});
  const requestId = useRef(0);
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [editing, setEditing] = useState<ContactLead>();
  const [msg, context] = message.useMessage();
  async function load(page = 1, values = query.current) {
    if (values.start && values.end && values.start.isAfter(values.end)) { msg.error("开始时间不能晚于结束时间"); return; }
    query.current = values;
    const id = ++requestId.current;
    setLoading(true);
    const params = new URLSearchParams({ page: String(page), pageSize: "20", contact: values.contact?.trim() || "" });
    if (values.start) params.set("start", values.start.toISOString());
    if (values.end) params.set("end", values.end.toISOString());
    try {
      const result = await api<Page<ContactLead>>(`leads?${params}`);
      if (id === requestId.current) setData(result);
    } catch (e) { if (id === requestId.current) msg.error((e as Error).message); }
    finally { if (id === requestId.current) setLoading(false); }
  }
  useEffect(() => { void load(); return () => { requestId.current++; }; }, []);
  return <Card title="留资记录" extra={<Button onClick={() => void load(data.page)}>刷新</Button>}>
    {context}
    <Form form={filters} layout="inline" onFinish={(values) => void load(1, values)} style={{ marginBottom: 20, rowGap: 12 }}>
      <Form.Item name="contact" label="联系方式"><Input allowClear maxLength={254} placeholder="邮箱或 WhatsApp" /></Form.Item>
      <Form.Item name="start" label="提交开始时间"><DatePicker showTime placeholder="开始时间" /></Form.Item>
      <Form.Item name="end" label="提交结束时间"><DatePicker showTime placeholder="结束时间" /></Form.Item>
      <Space><Button type="primary" htmlType="submit">查询</Button><Button onClick={() => { filters.resetFields(); void load(1, {}); }}>重置</Button></Space>
    </Form>
    <p>提交时间按最近提交时间筛选；筛选器使用浏览器本地时间，列表时间为北京时间。</p>
    <Table<ContactLead> rowKey="id" loading={loading} dataSource={data.items} scroll={{ x: 1300 }} pagination={{ current: data.page, pageSize: data.pageSize, total: data.total, showSizeChanger: false, onChange: (page) => void load(page) }} columns={[
      { title: "类型", dataIndex: "contactType", render: (value) => value === "email" ? "Email" : "WhatsApp" },
      { title: "联系方式", dataIndex: "contact", render: (value) => <Typography.Text copyable>{value}</Typography.Text> },
      { title: "首次提交时间", dataIndex: "firstSubmittedAt", render: displayTime },
      { title: "最近提交时间", dataIndex: "lastSubmittedAt", render: displayTime },
      { title: "提交次数", dataIndex: "submissionCount" },
      { title: "跟进状态", render: (_, row) => <Tag color={row.status === "pending" ? "blue" : row.status === "completed" ? "green" : "default"}>{statuses[row.status]}</Tag> },
      { title: "备注", dataIndex: "note", ellipsis: true },
      { title: "操作", render: (_, row) => canEdit && <Button onClick={() => { setEditing(row); form.setFieldsValue({ status: row.status, note: row.note }); }}>跟进</Button> },
    ]} />
    <Modal title="跟进留资" open={!!editing} confirmLoading={busy} onCancel={() => { if (!busy) setEditing(undefined); }} onOk={() => form.submit()} destroyOnHidden>
      <p>{editing?.contact}</p>
      <Form form={form} layout="vertical" onFinish={async (values) => {
        if (!editing || busy) return;
        setBusy(true);
        try {
          await api(`leads/${editing.id}`, "PATCH", { ...values, expectedVersion: editing.version });
          setEditing(undefined); msg.success("已保存"); await load(data.page);
        } catch (e) { msg.error((e as Error).message); }
        finally { setBusy(false); }
      }}>
        <Form.Item name="status" label="跟进状态" rules={[{ required: true }]}><Select options={Object.entries(statuses).map(([value, label]) => ({ value, label }))} /></Form.Item>
        <Form.Item name="note" label="跟进备注"><Input.TextArea rows={4} maxLength={2000} showCount /></Form.Item>
      </Form>
    </Modal>
  </Card>;
}
