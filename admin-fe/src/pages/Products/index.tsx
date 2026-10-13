import { useEffect, useRef, useState } from "react";
import { useModel } from "@umijs/max";
import {
  Button,
  Badge,
  Typography,
  Card,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  message,
} from "antd";
import {
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import ContentPage, {
  ContentHeading,
  formatTime,
} from "../../components/ContentPage";
import useContentList from "../../hooks/useContentList";
import { ProductView } from "@shop/content";
import { api } from "../../services/cms";
import { allows } from "../../services/auth";
import type {
  PublicCategory,
  Product,
  ProductSpec,
  ProductSKU,
  ProductSummary,
  PublicProduct,
} from "../../services/contracts";
import MediaSelect from "../../components/MediaSelect";
import RichTextEditor from "../../components/RichTextEditor";
function newKey() {
  return crypto.randomUUID();
}
function Specs({
  value = [],
  onChange,
}: {
  value?: ProductSpec[];
  onChange?: (value: ProductSpec[]) => void;
}) {
  function change(index: number, patch: Partial<ProductSpec>) {
    onChange?.(
      value.map((spec, j) => (j === index ? { ...spec, ...patch } : spec)),
    );
  }
  return (
    <Space orientation="vertical" style={{ width: "100%" }}>
      {value.map((spec, index) => (
        <Card
          size="small"
          key={spec.key}
          title={`规格维度 ${index + 1}`}
          extra={
            <Button
              danger
              onClick={() => onChange?.(value.filter((_, j) => j !== index))}
            >
              移除维度
            </Button>
          }
        >
          <Input
            placeholder="规格名称（英文，例如 Package size）"
            maxLength={100}
            value={spec.name}
            onChange={(event) => change(index, { name: event.target.value })}
          />
          <Space wrap style={{ marginTop: 12 }}>
            {spec.values.map((item, j) => (
              <Space key={item.key}>
                <Input
                  value={item.name}
                  placeholder="规格值（英文）"
                  maxLength={100}
                  onChange={(event) =>
                    change(index, {
                      values: spec.values.map((v, k) =>
                        k === j ? { ...v, name: event.target.value } : v,
                      ),
                    })
                  }
                />
                <Button
                  danger
                  onClick={() =>
                    change(index, {
                      values: spec.values.filter((_, k) => k !== j),
                    })
                  }
                >
                  移除
                </Button>
              </Space>
            ))}
            <Button
              disabled={spec.values.length >= 20}
              onClick={() =>
                change(index, {
                  values: [
                    ...spec.values,
                    { key: newKey(), name: "", sort: spec.values.length },
                  ],
                })
              }
            >
              新增规格值
            </Button>
          </Space>
        </Card>
      ))}
      <Button
        disabled={value.length >= 6}
        onClick={() =>
          onChange?.([
            ...value,
            {
              key: newKey(),
              name: "",
              sort: value.length,
              values: [{ key: newKey(), name: "", sort: 0 }],
            },
          ])
        }
      >
        新增规格维度
      </Button>
    </Space>
  );
}
const labels = { draft: "草稿", published: "已上架", offline: "已下架" };
export default function Products() {
  const { initialState } = useModel("@@initialState");
  const can = (p: string) => allows(initialState?.identity, p);
  const { data, loading, error, query, search, paginate, refresh } =
    useContentList<ProductSummary>("products");
  const [filterForm] = Form.useForm();
  const [modal, modalContext] = Modal.useModal();
  const [editing, setEditing] = useState<Product | null>();
  const [preview, setPreview] = useState<PublicProduct>();
  const [mobile, setMobile] = useState(false);
  const [priceEdit, setPriceEdit] = useState<{
    product: Product;
    sku: ProductSKU;
  }>();
  const [categories, setCategories] = useState<PublicCategory[]>([]);
  const [busy, setBusy] = useState(false);
  const [currencies, setCurrencies] = useState<{
    items: { code: string; minorUnit: number }[];
    defaultCurrency: string;
  }>({ items: [], defaultCurrency: "USD" });
  const [form] = Form.useForm<Product>();
  const cleanSnapshot = useRef("");
  const specs = Form.useWatch("specs", form) || [];
  const skus = Form.useWatch("skus", form) || [];
  const [msg, context] = message.useMessage();
  useEffect(() => {
    api<typeof currencies>("currencies")
      .then(setCurrencies)
      .catch((e: Error) => msg.error(e.message));
    api<{ items: PublicCategory[] }>("categories/options")
      .then((result) => setCategories(result.items))
      .catch((e: Error) => msg.error(e.message));
  }, []);
  function closeEditor() {
    if (busy) return;
    if (cleanSnapshot.current !== JSON.stringify(form.getFieldsValue(true))) {
      modal.confirm({
        title: "放弃未保存的产品修改？",
        okText: "放弃修改",
        cancelText: "继续编辑",
        onOk: () => setEditing(undefined),
      });
    } else setEditing(undefined);
  }
  async function edit(id?: string) {
    try {
      const product = id ? await api<Product>(`products/${id}`) : null;
      setEditing(product);
      form.resetFields();
      form.setFieldsValue(
        product || {
          status: "draft",
          detail: { html: "", mediaIds: [] },
          specs: [],
          skus: [
            {
              key: "default",
              enabled: true,
              sort: 0,
              specValueKeys: [],
              currency: currencies.defaultCurrency,
              salePrice: null,
            },
          ],
        },
      );
      cleanSnapshot.current = JSON.stringify(form.getFieldsValue(true));
    } catch (e) {
      msg.error((e as Error).message);
    }
  }
  function updateSKU(key: string, patch: Partial<ProductSKU>) {
    form.setFieldValue(
      "skus",
      skus.map((sku: ProductSKU) =>
        sku.key === key ? { ...sku, ...patch } : sku,
      ),
    );
  }
  function generate() {
    const count = specs.reduce(
      (count: number, spec: ProductSpec) => count * spec.values.length,
      1,
    );
    if (count > 500 || count === 0) {
      msg.error("请配置完整规格值；一次最多生成 500 个组合");
      return;
    }
    let combinations: string[][] = [[]];
    for (const spec of specs)
      combinations = combinations.flatMap((prior) =>
        spec.values.map((value: ProductSpec["values"][number]) => [
          ...prior,
          value.key,
        ]),
      );
    const signature = (keys: string[]) => [...keys].sort().join("|");
    form.setFieldValue(
      "skus",
      combinations.map((keys, index) => ({
        ...skus.find(
          (sku: ProductSKU) => signature(sku.specValueKeys) === signature(keys),
        ),
        key:
          skus.find(
            (sku: ProductSKU) =>
              signature(sku.specValueKeys) === signature(keys),
          )?.key || newKey(),
        specValueKeys: keys,
        enabled:
          skus.find(
            (sku: ProductSKU) =>
              signature(sku.specValueKeys) === signature(keys),
          )?.enabled ?? true,
        currency:
          skus.find(
            (sku: ProductSKU) =>
              signature(sku.specValueKeys) === signature(keys),
          )?.currency || currencies.defaultCurrency,
        salePrice:
          skus.find(
            (sku: ProductSKU) =>
              signature(sku.specValueKeys) === signature(keys),
          )?.salePrice ?? null,
        sort: index,
      })),
    );
  }
  function input() {
    const values = form.getFieldsValue(true);
    return {
      ...values,
      id: editing?.id || "",
      expectedVersion: editing?.version || 0,
      specs: values.specs || [],
      skus: (values.skus || []).map((sku: ProductSKU) => ({
        ...sku,
        salePrice: sku.salePrice?.trim() || null,
        price: undefined,
      })),
      detail: values.detail || { html: "", mediaIds: [] },
    };
  }
  async function save() {
    setBusy(true);
    try {
      await form.validateFields();
      const payload = input();
      const result = await api<Product>(
        editing ? `products/${editing.id}` : "products",
        editing ? "PUT" : "POST",
        payload,
      );
      msg.success("保存成功");
      setEditing(result);
      form.resetFields();
      form.setFieldsValue(result);
      cleanSnapshot.current = JSON.stringify(form.getFieldsValue(true));
      refresh();
    } catch (e) {
      msg.error((e as Error).message || "请检查必填项");
    } finally {
      setBusy(false);
    }
  }
  return (
    <ContentPage
      title="产品管理"
      description="维护产品资料、规格与官网上架状态"
      error={error}
      extra={
        can("product.create") && (
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => void edit()}
          >
            新增产品
          </Button>
        )
      }
      filters={
        <Form
          name="product-filters"
          form={filterForm}
          layout="vertical"
          onFinish={(values) =>
            search({
              q: values.q?.trim() || "",
              categoryId: values.categoryId || "",
              status: values.status || "",
            })
          }
        >
          <Form.Item name="q" label="产品名称">
            <Input allowClear maxLength={200} placeholder="输入产品名称" />
          </Form.Item>
          <Form.Item name="categoryId" label="产品分类">
            <Select
              allowClear
              showSearch
              optionFilterProp="label"
              placeholder="全部分类"
              options={categories.map((category) => ({
                value: category.id,
                label: category.name,
              }))}
            />
          </Form.Item>
          <Form.Item name="status" label="上架状态">
            <Select
              allowClear
              placeholder="全部状态"
              options={Object.entries(labels).map(([value, label]) => ({
                value,
                label,
              }))}
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
          产品列表{" "}
          <Typography.Text type="secondary">（{data.total}）</Typography.Text>
        </Typography.Title>
        <Button icon={<ReloadOutlined />} onClick={refresh} loading={loading}>
          刷新
        </Button>
      </div>
      <Table<ProductSummary>
        loading={loading}
        scroll={{ x: 1140 }}
        rowKey="id"
        dataSource={data.items}
        pagination={{
          current: query.page,
          pageSize: query.pageSize,
          total: data.total,
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 个产品`,
          onChange: paginate,
        }}
        columns={[
          {
            title: "产品名称",
            width: 260,
            render: (_, row) => (
              <div>
                <div className="content-cell-title">{row.name}</div>
                <div className="content-cell-meta">{row.slug}</div>
              </div>
            ),
          },
          {
            title: "分类",
            width: 180,
            dataIndex: "category",
            render: (value: string) => value || "未分配",
          },
          {
            title: "起价",
            width: 130,
            render: (_, row) => row.fromPrice?.displayPrice || "未定价",
          },
          {
            title: "状态",
            width: 100,
            render: (_, row) => (
              <Badge
                status={
                  row.status === "published"
                    ? "success"
                    : row.status === "offline"
                      ? "warning"
                      : "default"
                }
                text={row.status ? labels[row.status] : "—"}
              />
            ),
          },
          {
            title: "更新时间",
            width: 190,
            render: (_, row) => formatTime(row.updatedAt),
          },
          {
            title: "操作",
            width: 280,
            fixed: "right",
            render: (_, row) => (
              <Space wrap className="content-actions">
                {(can("product.update") || can("product.preview")) && (
                  <Button type="link" onClick={() => void edit(row.id)}>
                    {can("product.update") ? "编辑" : "查看"}
                  </Button>
                )}
                {row.status === "published" && (
                  <a
                    href={`${process.env.UMI_APP_WEBSITE_URL || (process.env.NODE_ENV === "development" ? "http://127.0.0.1:3000" : window.location.origin)}/products/${encodeURIComponent(row.slug)}`}
                    target="_blank"
                    rel="noreferrer"
                  >
                    查看官网
                  </a>
                )}
                {can("product.publish") && (
                  <Popconfirm
                    title={`确认${row.status === "published" ? "下架" : "上架"}此产品？`}
                    description={row.name}
                    onConfirm={async () => {
                      try {
                        await api(`products/${row.id}/status`, "PATCH", {
                          status:
                            row.status === "published"
                              ? "offline"
                              : "published",
                          expectedVersion: row.version,
                        });
                        msg.success("上架状态已更新");
                        refresh();
                      } catch (e) {
                        msg.error((e as Error).message);
                        throw e;
                      }
                    }}
                  >
                    <Button type="link">
                      {row.status === "published" ? "下架" : "上架"}
                    </Button>
                  </Popconfirm>
                )}
                {can("product.delete") && (
                  <Popconfirm
                    title="确认删除产品？"
                    onConfirm={async () => {
                      try {
                        await api(`products/${row.id}`, "DELETE", {
                          expectedVersion: row.version,
                        });
                        refresh();
                      } catch (e) {
                        msg.error((e as Error).message);
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
        title={editing ? `编辑产品 · ${editing.name}` : "新增产品"}
        open={editing !== undefined}
        onCancel={closeEditor}
        width={1200}
        footer={
          <Space>
            {can("product.preview") && (
              <Button
                loading={busy}
                onClick={async () => {
                  setBusy(true);
                  try {
                    setPreview(
                      await api<PublicProduct>(
                        "products/preview",
                        "POST",
                        input(),
                      ),
                    );
                  } catch (e) {
                    msg.error((e as Error).message);
                  } finally {
                    setBusy(false);
                  }
                }}
              >
                预览未保存内容
              </Button>
            )}
            {can(editing ? "product.update" : "product.create") && (
              <Button type="primary" loading={busy} onClick={() => void save()}>
                保存
              </Button>
            )}
            <Button disabled={busy} onClick={closeEditor}>
              关闭
            </Button>
          </Space>
        }
        destroyOnHidden
      >
        <Form
          name="product-editor"
          form={form}
          layout="vertical"
          preserve={false}
          disabled={busy || !can(editing ? "product.update" : "product.create")}
        >
          <ContentHeading
            title="基础信息"
            description="产品名称和简介用于英文官网展示。"
          />
          <div className="content-form-grid">
            <Form.Item
              name="name"
              label="产品名称（英文）"
              rules={[{ required: true, max: 200 }]}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="categoryId"
              label="商品分类"
              rules={[{ required: true, message: "请选择商品分类" }]}
            >
              <Select
                showSearch
                optionFilterProp="label"
                placeholder="请选择分类"
                options={categories.map((c) => ({
                  value: c.id,
                  label: c.name,
                }))}
              />
            </Form.Item>
            <Form.Item
              className="full-width"
              name="summary"
              label="简介（英文）"
            >
              <Input.TextArea rows={3} />
            </Form.Item>
          </div>
          <ContentHeading title="产品媒体" />
          <div className="content-form-grid">
            <Form.Item
              name="imageMediaId"
              label="产品图片"
              rules={[{ required: true, message: "请选择产品图片" }]}
            >
              <MediaSelect types={["image"]} />
            </Form.Item>
            <Form.Item name="videoMediaId" label="产品视频（选填）">
              <MediaSelect types={["video"]} />
            </Form.Item>
            <Form.Item name="reportMediaId" label="检测报告图片（选填）">
              <MediaSelect types={["image"]} />
            </Form.Item>
          </div>
          <ContentHeading
            title="规格与价格"
            description="设置规格维度后生成组合，再填写各组合的售价。"
          />
          <Form.Item name="specs" label="自定义规格（修改后请更新 SKU 组合）">
            <Specs />
          </Form.Item>
          <Form.Item name="skus" hidden>
            <Input />
          </Form.Item>
          <Space>
            <Button onClick={generate}>生成规格组合</Button>
            <Button
              disabled={!specs.length || skus.length >= 500}
              onClick={() =>
                form.setFieldValue("skus", [
                  ...skus,
                  {
                    key: newKey(),
                    enabled: true,
                    sort: skus.length,
                    specValueKeys: specs.map(
                      (spec: ProductSpec) => spec.values[0]?.key || "",
                    ),
                    currency: currencies.defaultCurrency,
                    salePrice: null,
                  },
                ])
              }
            >
              手动新增组合
            </Button>
          </Space>
          <Table<ProductSKU>
            scroll={{ x: 750 }}
            rowKey="key"
            dataSource={skus}
            pagination={{ pageSize: 20 }}
            columns={[
              {
                title: "规格组合",
                render: (_, sku, index) =>
                  specs.length ? (
                    <Space wrap>
                      {specs.map((spec: ProductSpec) => (
                        <Select
                          key={spec.key}
                          style={{ minWidth: 130 }}
                          placeholder={spec.name}
                          value={sku.specValueKeys.find((key) =>
                            spec.values.some((value) => value.key === key),
                          )}
                          options={spec.values.map((value) => ({
                            value: value.key,
                            label: `${spec.name}: ${value.name}`,
                          }))}
                          onChange={(value) =>
                            updateSKU(sku.key, {
                              specValueKeys: [
                                ...sku.specValueKeys.filter(
                                  (key) =>
                                    !spec.values.some(
                                      (item) => item.key === key,
                                    ),
                                ),
                                value,
                              ],
                            })
                          }
                        />
                      ))}
                    </Space>
                  ) : (
                    "默认 SKU"
                  ),
              },
              {
                title: "币种",
                render: (_, sku, index) => (
                  <Select
                    disabled={!!sku.id && !can("product.price.update")}
                    value={sku.currency}
                    options={currencies.items.map((item) => ({
                      value: item.code,
                      label: item.code,
                    }))}
                    onChange={(value) =>
                      updateSKU(sku.key, { currency: value })
                    }
                  />
                ),
              },
              {
                title: "售价",
                render: (_, sku, index) => (
                  <Input
                    disabled={!can("product.price.update")}
                    value={sku.salePrice || ""}
                    placeholder="草稿可不定价"
                    onChange={(event) =>
                      updateSKU(sku.key, {
                        salePrice: event.target.value || null,
                      })
                    }
                  />
                ),
              },
              {
                title: "启用",
                render: (_, sku, index) => (
                  <Switch
                    checked={sku.enabled}
                    onChange={(value) => updateSKU(sku.key, { enabled: value })}
                  />
                ),
              },
              {
                title: "操作",
                render: (_, sku, index) => (
                  <Space>
                    {!!editing && !!sku.id && can("product.price.update") && (
                      <Button
                        onClick={() => {
                          if (
                            cleanSnapshot.current !==
                            JSON.stringify(form.getFieldsValue(true))
                          ) {
                            msg.warning("请先保存产品修改，再独立改价");
                            return;
                          }
                          setPriceEdit({ product: editing, sku });
                        }}
                      >
                        独立改价
                      </Button>
                    )}
                    {specs.length > 0 && (
                      <Button
                        danger
                        onClick={() =>
                          form.setFieldValue(
                            "skus",
                            skus.filter(
                              (item: ProductSKU) => item.key !== sku.key,
                            ),
                          )
                        }
                      >
                        移除
                      </Button>
                    )}
                  </Space>
                ),
              },
            ]}
          />
          <ContentHeading title="产品详情" />
          <Form.Item name="detail" label="产品详情（英文，媒体从库选择）">
            <RichTextEditor />
          </Form.Item>
          <ContentHeading title="发布设置" />
          <div className="content-form-grid">
            <Form.Item name="status" label="状态">
              <Select
                disabled={!can("product.publish")}
                options={Object.entries(labels).map(([value, label]) => ({
                  value,
                  label,
                }))}
              />
            </Form.Item>
          </div>
        </Form>
      </Modal>
      <Modal
        title="产品预览（不保存）"
        open={!!preview}
        onCancel={() => setPreview(undefined)}
        width={1050}
        footer={null}
      >
        <Space>
          <Button onClick={() => setMobile(false)}>桌面</Button>
          <Button onClick={() => setMobile(true)}>手机</Button>
        </Space>
        <div
          style={{
            width: mobile ? 375 : "100%",
            maxWidth: "100%",
            margin: "20px auto",
          }}
        >
          {preview && (
            <ProductView
              key={JSON.stringify(preview)}
              product={preview}
              mediaPrefix="/backend"
            />
          )}
        </div>
      </Modal>
      <Modal
        title="独立修改 SKU 价格"
        open={!!priceEdit}
        onCancel={() => setPriceEdit(undefined)}
        footer={null}
        destroyOnHidden
      >
        <Form
          layout="vertical"
          initialValues={{ salePrice: priceEdit?.sku.salePrice }}
          onFinish={async (values) => {
            if (!priceEdit) return;
            try {
              const result = await api<Product>(
                `products/${priceEdit.product.id}/skus/${priceEdit.sku.id}/price`,
                "PATCH",
                {
                  salePrice: values.salePrice,
                  currency: priceEdit.sku.currency,
                  expectedVersion: priceEdit.product.version,
                },
              );
              setPriceEdit(undefined);
              setEditing(result);
              form.resetFields();
              form.setFieldsValue(result);
              cleanSnapshot.current = JSON.stringify(form.getFieldsValue(true));
              refresh();
            } catch (e) {
              msg.error((e as Error).message);
            }
          }}
        >
          <Form.Item
            name="salePrice"
            label={`价格（${priceEdit?.sku.currency || "USD"}）`}
            rules={[{ required: true, pattern: /^\d+(?:\.\d+)?$/ }]}
          >
            <Input />
          </Form.Item>
          <Button type="primary" htmlType="submit">
            保存价格
          </Button>
        </Form>
      </Modal>
    </ContentPage>
  );
}
