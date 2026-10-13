import { useEffect, useState } from "react";
import { useModel } from "@umijs/max";
import {
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Space,
  Table,
  Switch,
  Tag,
  message,
} from "antd";
import { api } from "../../services/cms";
import { allows } from "../../services/auth";
import type { FAQ, Page } from "../../services/contracts";

export default function FAQs() {
  const { initialState } = useModel("@@initialState");
  const can = (p: string) => allows(initialState?.identity, p);
  const [data, setData] = useState<Page<FAQ>>({
    items: [],
    total: 0,
    page: 1,
    pageSize: 20,
  });
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<FAQ>();
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm<FAQ>();
  const [msg, context] = message.useMessage();
  async function load(page = data.page, q = query) {
    setLoading(true);
    try {
      setData(
        await api<Page<FAQ>>(`faqs?page=${page}&q=${encodeURIComponent(q)}`),
      );
    } catch (e) {
      msg.error((e as Error).message);
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    void load(1);
  }, []);
  async function edit(row?: FAQ) {
    try {
      const current = row ? await api<FAQ>(`faqs/${row.id}`) : undefined;
      setEditing(current);
      form.resetFields();
      form.setFieldsValue(
        current || { question: "", answer: "", enabled: false, sort: 0 },
      );
      setOpen(true);
    } catch (e) {
      msg.error((e as Error).message);
    }
  }
  return (
    <Card
      title="问答配置"
      extra={
        can("faq.create") && (
          <Button type="primary" onClick={() => void edit()}>
            新增问答
          </Button>
        )
      }
    >
      {context}
      <Input.Search
        placeholder="按问题搜索"
        allowClear
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onSearch={(q) => void load(1, q)}
        style={{ maxWidth: 360, marginBottom: 20 }}
      />
      <Table
        rowKey="id"
        dataSource={data.items}
        loading={loading}
        pagination={{
          current: data.page,
          pageSize: data.pageSize,
          total: data.total,
          showSizeChanger: false,
          onChange: (page) => void load(page),
        }}
        columns={[
          { title: "问题", dataIndex: "question" },
          { title: "答案", dataIndex: "answer", ellipsis: true },
          { title: "排序", dataIndex: "sort" },
          {
            title: "是否开启",
            dataIndex: "enabled",
            render: (enabled: boolean) => (
              <Tag color={enabled ? "green" : "default"}>
                {enabled ? "开启" : "关闭"}
              </Tag>
            ),
          },
          {
            title: "操作",
            render: (_, row) => (
              <Space>
                {can("faq.update") && (
                  <Button onClick={() => void edit(row)}>编辑</Button>
                )}
                {can("faq.delete") && (
                  <Popconfirm
                    title="删除此问答？删除后官网不再展示。"
                    onConfirm={async () => {
                      try {
                        await api(`faqs/${row.id}`, "DELETE", {
                          expectedVersion: row.version,
                        });
                        msg.success("已删除");
                        await load(
                          data.items.length === 1 && data.page > 1
                            ? data.page - 1
                            : data.page,
                        );
                      } catch (e) {
                        msg.error((e as Error).message);
                      }
                    }}
                  >
                    <Button danger>删除</Button>
                  </Popconfirm>
                )}
              </Space>
            ),
          },
        ]}
      />
      <Modal
        title={editing ? "编辑问答" : "新增问答"}
        open={open}
        confirmLoading={busy}
        onCancel={() => setOpen(false)}
        onOk={() => form.submit()}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={async (values) => {
            setBusy(true);
            try {
              await api(
                editing ? `faqs/${editing.id}` : "faqs",
                editing ? "PUT" : "POST",
                { ...values, expectedVersion: editing?.version || 0 },
              );
              setOpen(false);
              msg.success("已保存");
              await load();
            } catch (e) {
              msg.error((e as Error).message);
            } finally {
              setBusy(false);
            }
          }}
        >
          <Form.Item
            name="question"
            label="问题（英文）"
            rules={[{ required: true, whitespace: true, max: 500 }]}
          >
            <Input maxLength={500} />
          </Form.Item>
          <Form.Item
            name="answer"
            label="答案（英文）"
            rules={[{ required: true, whitespace: true, max: 10000 }]}
          >
            <Input.TextArea rows={6} maxLength={10000} showCount />
          </Form.Item>
          <Form.Item
            name="enabled"
            label="是否开启"
            valuePropName="checked"
            extra="开启后官网展示此问答"
          >
            <Switch checkedChildren="开启" unCheckedChildren="关闭" />
          </Form.Item>
          <Form.Item
            name="sort"
            label="排序（数值越小越靠前）"
            rules={[{ required: true }]}
          >
            <InputNumber precision={0} min={-2147483648} max={2147483647} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
