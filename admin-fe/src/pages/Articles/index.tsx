import { useRef, useState } from "react";
import { useModel } from "@umijs/max";
import {
  Badge,
  Button,
  Form,
  Image,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import {
  FileTextOutlined,
  PlusOutlined,
  PushpinFilled,
  ReloadOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import { ArticleView } from "@shop/content";
import { api, assetURL, type Article } from "../../services/cms";
import { allows } from "../../services/auth";
import type { PublicArticle } from "../../services/contracts";
import RichTextEditor from "../../components/RichTextEditor";
import MediaSelect from "../../components/MediaSelect";
import ContentPage, {
  ContentHeading,
  formatTime,
} from "../../components/ContentPage";
import useContentList from "../../hooks/useContentList";

const labels = { draft: "草稿", published: "已发布", offline: "已下线" };
type ArticleSummary = Pick<
  Article,
  | "id"
  | "title"
  | "slug"
  | "summary"
  | "cover"
  | "firstPublishedAt"
  | "status"
  | "pinned"
  | "sort"
  | "version"
  | "updatedAt"
>;

export default function ArticlesPage() {
  const { initialState } = useModel("@@initialState");
  const can = (p: string) => allows(initialState?.identity, p);
  const { data, loading, error, query, search, paginate, refresh } =
    useContentList<ArticleSummary>("articles");
  const [editing, setEditing] = useState<Article>();
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [previewBusy, setPreviewBusy] = useState(false);
  const [preview, setPreview] = useState<PublicArticle>();
  const [mobile, setMobile] = useState(false);
  const [form] = Form.useForm();
  const [filterForm] = Form.useForm();
  const cleanSnapshot = useRef("");
  const [msg, context] = message.useMessage();
  const [modal, modalContext] = Modal.useModal();

  async function edit(row?: ArticleSummary) {
    try {
      const article = row
        ? await api<Article>(`articles/${row.id}`)
        : undefined;
      setEditing(article);
      form.resetFields();
      form.setFieldsValue(
        article || {
          title: "",
          summary: "",
          body: { html: "", mediaIds: [] },
          coverMediaId: "",
          status: "draft",
          pinned: false,
          sort: 0,
        },
      );
      cleanSnapshot.current = JSON.stringify(form.getFieldsValue(true));
      setOpen(true);
    } catch (e) {
      msg.error((e as Error).message);
    }
  }
  function closeEditor() {
    if (busy || previewBusy) return;
    if (cleanSnapshot.current !== JSON.stringify(form.getFieldsValue(true))) {
      modal.confirm({
        title: "放弃未保存的新闻修改？",
        okText: "放弃修改",
        cancelText: "继续编辑",
        onOk: () => setOpen(false),
      });
    } else setOpen(false);
  }
  const payload = (values: Record<string, unknown>) => ({
    ...values,
    status: editing?.status || "draft",
    firstPublishedAt: editing?.firstPublishedAt || null,
    expectedVersion: editing?.version || 0,
  });
  async function status(article: ArticleSummary) {
    try {
      await api(`articles/${article.id}/status`, "PATCH", {
        status: article.status === "published" ? "offline" : "published",
        expectedVersion: article.version,
      });
      refresh();
      msg.success("发布状态已更新");
    } catch (e) {
      msg.error((e as Error).message);
      throw e;
    }
  }
  async function save(values: Record<string, unknown>) {
    setBusy(true);
    try {
      await api(
        editing ? `articles/${editing.id}` : "articles",
        editing ? "PUT" : "POST",
        payload(values),
      );
      setOpen(false);
      refresh();
      msg.success("新闻已保存");
    } catch (e) {
      msg.error((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  async function showPreview() {
    setPreviewBusy(true);
    try {
      const values = await form.validateFields();
      setPreview(await api("articles/preview", "POST", payload(values)));
    } catch (e) {
      if (e instanceof Error) msg.error(e.message);
    } finally {
      setPreviewBusy(false);
    }
  }
  return (
    <ContentPage
      title="新闻管理"
      description="管理官网 News 内容、发布状态与展示顺序"
      error={error}
      extra={
        can("article.create") && (
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => void edit()}
          >
            新增新闻
          </Button>
        )
      }
      filters={
        <Form
          name="news-filters"
          form={filterForm}
          layout="vertical"
          onFinish={(values) =>
            search({
              q: values.q?.trim() || "",
              status: values.status || "",
              pinned: values.pinned || "",
            })
          }
        >
          <Form.Item name="q" label="新闻标题">
            <Input allowClear maxLength={200} placeholder="输入新闻标题" />
          </Form.Item>
          <Form.Item name="status" label="发布状态">
            <Select
              allowClear
              placeholder="全部状态"
              options={Object.entries(labels).map(([value, label]) => ({
                value,
                label,
              }))}
            />
          </Form.Item>
          <Form.Item name="pinned" label="置顶状态">
            <Select
              allowClear
              placeholder="全部新闻"
              options={[
                { value: "true", label: "已置顶" },
                { value: "false", label: "未置顶" },
              ]}
            />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button
                type="primary"
                htmlType="submit"
                icon={<SearchOutlined />}
              >
                查询
              </Button>
              <Button
                onClick={() => {
                  filterForm.resetFields();
                  search({});
                }}
              >
                重置
              </Button>
            </Space>
          </Form.Item>
        </Form>
      }
    >
      {context}
      {modalContext}
      <div className="content-toolbar">
        <Typography.Title level={5}>
          新闻列表{" "}
          <Typography.Text type="secondary">（{data.total}）</Typography.Text>
        </Typography.Title>
        <Button icon={<ReloadOutlined />} onClick={refresh} loading={loading}>
          刷新
        </Button>
      </div>
      <Table<ArticleSummary>
        rowKey="id"
        loading={loading}
        dataSource={data.items}
        scroll={{ x: 1200 }}
        pagination={{
          current: query.page,
          pageSize: query.pageSize,
          total: data.total,
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 篇新闻`,
          onChange: paginate,
        }}
        columns={[
          {
            title: "新闻",
            width: 350,
            render: (_, article) => (
              <div className="content-cover">
                {article.cover ? (
                  <Image
                    width={80}
                    height={54}
                    src={assetURL(article.cover)}
                    alt={article.title}
                  />
                ) : (
                  <span className="content-cover-placeholder">
                    <FileTextOutlined />
                  </span>
                )}
                <div>
                  <div className="content-cell-title">
                    {article.pinned && (
                      <PushpinFilled
                        style={{ color: "#d97706", marginRight: 6 }}
                      />
                    )}
                    {article.title}
                  </div>
                  <div className="content-cell-meta">/news/{article.slug}</div>
                </div>
              </div>
            ),
          },
          {
            title: "状态",
            width: 110,
            render: (_, article) => (
              <Badge
                status={
                  article.status === "published"
                    ? "success"
                    : article.status === "offline"
                      ? "warning"
                      : "default"
                }
                text={labels[article.status]}
              />
            ),
          },
          {
            title: "置顶",
            width: 90,
            render: (_, article) =>
              article.pinned ? (
                <Tag color="gold">已置顶</Tag>
              ) : (
                <Typography.Text type="secondary">—</Typography.Text>
              ),
          },
          { title: "排序", width: 80, dataIndex: "sort" },
          {
            title: "首次发布时间",
            width: 170,
            render: (_, article) => formatTime(article.firstPublishedAt),
          },
          {
            title: "更新时间",
            width: 170,
            render: (_, article) => formatTime(article.updatedAt),
          },
          {
            title: "操作",
            width: 260,
            fixed: "right",
            render: (_, article) => (
              <Space wrap className="content-actions">
                {(can("article.update") || can("article.preview")) && (
                  <Button type="link" onClick={() => void edit(article)}>
                    {can("article.update") ? "编辑" : "查看"}
                  </Button>
                )}
                {article.status === "published" && (
                  <a
                    href={`${process.env.UMI_APP_WEBSITE_URL || (process.env.NODE_ENV === "development" ? "http://127.0.0.1:3000" : window.location.origin)}/news/${encodeURIComponent(article.slug)}`}
                    target="_blank"
                    rel="noreferrer"
                  >
                    查看官网
                  </a>
                )}
                {can("article.publish") && (
                  <Popconfirm
                    title={`确认${article.status === "published" ? "下线" : "发布"}此新闻？`}
                    description={article.title}
                    onConfirm={() => status(article)}
                  >
                    <Button type="link">
                      {article.status === "published" ? "下线" : "发布"}
                    </Button>
                  </Popconfirm>
                )}
                {can("article.delete") && (
                  <Popconfirm
                    title="确认删除新闻？"
                    description={article.title}
                    onConfirm={async () => {
                      try {
                        await api(`articles/${article.id}`, "DELETE", {
                          expectedVersion: article.version,
                        });
                        refresh();
                        msg.success("新闻已删除");
                      } catch (e) {
                        msg.error((e as Error).message);
                        throw e;
                      }
                    }}
                  >
                    <Button type="link" danger>
                      删除
                    </Button>
                  </Popconfirm>
                )}
              </Space>
            ),
          },
        ]}
      />
      <Modal
        className="content-editor"
        title={editing ? `编辑新闻 · ${editing.title}` : "新增新闻"}
        open={open}
        width={1100}
        destroyOnHidden
        onCancel={closeEditor}
        footer={
          <Space>
            <Button disabled={busy || previewBusy} onClick={closeEditor}>
              关闭
            </Button>
            {can("article.preview") && (
              <Button
                loading={previewBusy}
                disabled={busy}
                onClick={() => void showPreview()}
              >
                预览未保存内容
              </Button>
            )}
            {can(editing ? "article.update" : "article.create") && (
              <Button
                type="primary"
                loading={busy}
                disabled={previewBusy}
                onClick={() => form.submit()}
              >
                保存新闻
              </Button>
            )}
          </Space>
        }
      >
        <Form
          name="news-editor"
          form={form}
          layout="vertical"
          onFinish={save}
          disabled={
            busy ||
            previewBusy ||
            !can(editing ? "article.update" : "article.create")
          }
        >
          <ContentHeading
            title="基础信息"
            description="英文标题、摘要与封面用于官网新闻列表。"
          />
          <div className="content-form-grid">
            <Form.Item
              name="title"
              label="新闻标题（英文）"
              rules={[{ required: true, whitespace: true, max: 200 }]}
            >
              <Input maxLength={200} showCount />
            </Form.Item>
            <Form.Item
              className="full-width"
              name="summary"
              label="摘要（英文）"
            >
              <Input.TextArea rows={3} maxLength={2000} showCount />
            </Form.Item>
            <Form.Item
              className="full-width"
              name="coverMediaId"
              label="封面图片"
            >
              <MediaSelect />
            </Form.Item>
          </div>
          <ContentHeading
            title="新闻正文"
            description="支持富文本排版与媒体库素材，保存前可预览官网效果。"
          />
          <Form.Item name="body" label="正文（英文）">
            <RichTextEditor />
          </Form.Item>
          <ContentHeading
            title="展示设置"
            description="置顶新闻优先展示，同组内排序值越小越靠前；发布与下线在列表中操作。"
          />
          <div className="content-form-grid">
            <Form.Item name="pinned" label="置顶" valuePropName="checked">
              <Switch checkedChildren="置顶" unCheckedChildren="普通" />
            </Form.Item>
            <Form.Item name="sort" label="排序">
              <InputNumber precision={0} min={-2147483648} max={2147483647} />
            </Form.Item>
          </div>
        </Form>
      </Modal>
      <Modal
        title="新闻官网预览（未保存）"
        open={!!preview}
        width={1100}
        footer={null}
        onCancel={() => setPreview(undefined)}
      >
        <Space>
          <Button
            type={!mobile ? "primary" : "default"}
            onClick={() => setMobile(false)}
          >
            桌面
          </Button>
          <Button
            type={mobile ? "primary" : "default"}
            onClick={() => setMobile(true)}
          >
            手机
          </Button>
        </Space>
        <div style={{ maxWidth: mobile ? 390 : 1000, margin: "24px auto" }}>
          {preview && <ArticleView article={preview} mediaPrefix="/backend" />}
        </div>
      </Modal>
    </ContentPage>
  );
}
