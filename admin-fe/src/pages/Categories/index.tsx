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
  message,
} from "antd";
import { api } from "../../services/cms";
import { allows } from "../../services/auth";
import type { Category, Page } from "../../services/contracts";

export default function Categories() {
  const { initialState } = useModel("@@initialState");
  const can = (p: string) => allows(initialState?.identity, p);
  const [data, setData] = useState<Page<Category>>({
    items: [],
    total: 0,
    page: 1,
    pageSize: 20,
  });
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<Category>();
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm<Category>();
  const [msg, context] = message.useMessage();
  async function load(page = data.page, q = query) {
    setLoading(true);
    try {
      setData(
        await api<Page<Category>>(
          `categories?page=${page}&q=${encodeURIComponent(q)}`,
        ),
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
  async function edit(row?: Category) {
    try {
      const current = row
        ? await api<Category>(`categories/${row.id}`)
        : undefined;
      setEditing(current);
      form.resetFields();
      form.setFieldsValue(current || { name: "", sort: 0 });
      setOpen(true);
    } catch (e) {
      msg.error((e as Error).message);
    }
  }
  return (
    <Card
      title="商品分类"
      extra={
        can("category.create") && (
          <Button type="primary" onClick={() => void edit()}>
            新增分类
          </Button>
        )
      }
    >
      {context}
      <Input.Search
        placeholder="按分类名称搜索"
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
          { title: "分类名称", dataIndex: "name" },
          { title: "排序", dataIndex: "sort" },
          {
            title: "操作",
            render: (_, row) => (
              <Space>
                {can("category.update") && (
                  <Button onClick={() => void edit(row)}>编辑</Button>
                )}
                {can("category.delete") && (
                  <Popconfirm
                    title="删除此分类？被商品使用的分类无法删除。"
                    onConfirm={async () => {
                      try {
                        await api(`categories/${row.id}`, "DELETE", {
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
        title={editing ? "编辑分类" : "新增分类"}
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
                editing ? `categories/${editing.id}` : "categories",
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
            name="name"
            label="分类名称（英文）"
            rules={[{ required: true, whitespace: true, max: 200 }]}
          >
            <Input maxLength={200} />
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
