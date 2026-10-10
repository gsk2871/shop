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
  Switch,
  Table,
  Tag,
  message,
} from "antd";
import { ArticleView } from "@shop/content";
import { api, type Article } from "../../services/cms";
import { allows } from "../../services/auth";
import type { Page, PublicArticle } from "../../services/contracts";
import RichTextEditor from "../../components/RichTextEditor";
import MediaSelect from "../../components/MediaSelect";
export default function ArticlesPage() {
  const { initialState } = useModel("@@initialState");
  const can = (p: string) => allows(initialState?.identity, p);
  const [data, setData] = useState<Page<Article>>({
    items: [],
    total: 0,
    page: 1,
    pageSize: 20,
  });
  const [editing, setEditing] = useState<Article>();
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [preview, setPreview] = useState<PublicArticle>();
  const [mobile, setMobile] = useState(false);
  const [form] = Form.useForm();
  const [msg, context] = message.useMessage();
  async function load(page = data.page) {
    try {
      setData(await api(`articles?page=${page}&pageSize=20`));
    } catch (e) {
      msg.error((e as Error).message);
    }
  }
  useEffect(() => {
    void load(1);
  }, []);
  async function edit(row?: Article) {
    try {
      const a = row ? await api<Article>(`articles/${row.id}`) : undefined;
      setEditing(a);
      form.resetFields();
      form.setFieldsValue(
        a || {
          title: "",
          slug: "",
          summary: "",
          body: { html: "", mediaIds: [] },
          coverMediaId: "",
          status: "draft",
          seoTitle: "",
          seoDescription: "",
          pinned: false,
          sort: 0,
        },
      );
      setOpen(true);
    } catch (e) {
      msg.error((e as Error).message);
    }
  }
  const payload = (values: Record<string, unknown>) => ({
    ...values,
    status: editing?.status || "draft",
    firstPublishedAt: editing?.firstPublishedAt || null,
    expectedVersion: editing?.version || 0,
  });
  async function status(a: Article) {
    try {
      await api(`articles/${a.id}/status`, "PATCH", {
        status: a.status === "published" ? "offline" : "published",
        expectedVersion: a.version,
      });
      await load();
      msg.success("发布状态已更新");
    } catch (e) {
      msg.error((e as Error).message);
    }
  }
  return (
    <Card
      title="文章管理"
      extra={
        can("article.create") && (
          <Button type="primary" onClick={() => void edit()}>
            新增文章
          </Button>
        )
      }
    >
      {context}
      <Table<Article>
        rowKey="id"
        dataSource={data.items}
        pagination={{
          current: data.page,
          pageSize: data.pageSize,
          total: data.total,
          onChange: (p) => void load(p),
        }}
        columns={[
          { title: "英文标题", dataIndex: "title" },
          { title: "地址", dataIndex: "slug" },
          {
            title: "状态",
            render: (_, a) => (
              <Tag>
                {
                  { draft: "草稿", published: "已发布", offline: "已下线" }[
                    a.status
                  ]
                }
              </Tag>
            ),
          },
          { title: "首次发布时间", dataIndex: "firstPublishedAt" },
          {
            title: "操作",
            render: (_, a) => (
              <Space>
                {can("article.update") && (
                  <Button onClick={() => void edit(a)}>编辑</Button>
                )}
                {can("article.publish") && (
                  <Popconfirm
                    title="确认改变发布状态？"
                    onConfirm={() => status(a)}
                  >
                    <Button>
                      {a.status === "published" ? "下线" : "发布"}
                    </Button>
                  </Popconfirm>
                )}
                {can("article.delete") && (
                  <Popconfirm
                    title="确认删除文章？"
                    onConfirm={async () => {
                      try {
                        await api(`articles/${a.id}`, "DELETE", {
                          expectedVersion: a.version,
                        });
                        await load();
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
        title={editing ? "编辑文章" : "新增文章"}
        open={open}
        width={960}
        destroyOnHidden
        onCancel={() => setOpen(false)}
        confirmLoading={busy}
        onOk={() => form.submit()}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={async (values) => {
            setBusy(true);
            try {
              await api(
                editing ? `articles/${editing.id}` : "articles",
                editing ? "PUT" : "POST",
                payload(values),
              );
              setOpen(false);
              await load();
              msg.success("已保存");
            } catch (e) {
              msg.error((e as Error).message);
            } finally {
              setBusy(false);
            }
          }}
        >
          <Form.Item
            name="title"
            label="英文标题"
            rules={[{ required: true, max: 200 }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="slug"
            label="地址（英文小写、数字、连字符）"
            rules={[
              {
                required: true,
                max: 120,
                pattern: /^[a-z0-9]+(?:-[a-z0-9]+)*$/,
              },
            ]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="summary" label="英文摘要">
            <Input.TextArea maxLength={2000} />
          </Form.Item>
          <Form.Item name="coverMediaId" label="封面图片">
            <MediaSelect />
          </Form.Item>
          <Form.Item name="body" label="英文正文">
            <RichTextEditor />
          </Form.Item>
          <Space>
            <Form.Item name="pinned" label="置顶" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="sort" label="排序（较小优先）">
              <InputNumber precision={0} />
            </Form.Item>
          </Space>
          <Form.Item name="seoTitle" label="英文 SEO 标题">
            <Input maxLength={200} />
          </Form.Item>
          <Form.Item name="seoDescription" label="英文 SEO 描述">
            <Input.TextArea maxLength={5000} />
          </Form.Item>
          {can("article.preview") && (
            <Button
              onClick={async () => {
                try {
                  const v = await form.validateFields();
                  setPreview(await api("articles/preview", "POST", payload(v)));
                } catch (e) {
                  msg.error((e as Error).message);
                }
              }}
            >
              预览当前未保存内容
            </Button>
          )}
        </Form>
      </Modal>
      <Modal
        title="官网预览（未保存）"
        open={!!preview}
        width={1100}
        footer={null}
        onCancel={() => setPreview(undefined)}
      >
        <Space>
          手机宽度
          <Switch checked={mobile} onChange={setMobile} />
        </Space>
        <div style={{ maxWidth: mobile ? 390 : 1000, margin: "24px auto" }}>
          {preview && <ArticleView article={preview} mediaPrefix="/backend" />}
        </div>
      </Modal>
    </Card>
  );
}
