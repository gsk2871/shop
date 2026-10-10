import { useEffect, useState } from "react";
import { useModel } from "@umijs/max";
import {
  Button,
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
import { ProductView } from "@shop/content";
import { api } from "../../services/cms";
import { allows } from "../../services/auth";
import type {
  Page,
  Product,
  ProductSpec,
  ProductSKU,
  ProductSummary,
  PublicProduct,
} from "../../services/contracts";
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
  const [data, setData] = useState<Page<ProductSummary>>({
    items: [],
    total: 0,
    page: 1,
    pageSize: 20,
  });
  const [editing, setEditing] = useState<Product | null>();
  const [preview, setPreview] = useState<PublicProduct>();
  const [mobile, setMobile] = useState(false);
  const [priceEdit, setPriceEdit] = useState<{
    product: Product;
    sku: ProductSKU;
  }>();
  const [query, setQuery] = useState("");
  const [busy, setBusy] = useState(false);
  const [currencies, setCurrencies] = useState<{
    items: { code: string; minorUnit: number }[];
    defaultCurrency: string;
  }>({ items: [], defaultCurrency: "USD" });
  const [form] = Form.useForm<Product>();
  const specs = Form.useWatch("specs", form) || [];
  const skus = Form.useWatch("skus", form) || [];
  const [msg, context] = message.useMessage();
  async function load(page = data.page, pageSize = data.pageSize, q = query) {
    setBusy(true);
    try {
      setData(
        await api(
          `products?page=${page}&pageSize=${pageSize}&q=${encodeURIComponent(q)}`,
        ),
      );
    } catch (e) {
      msg.error((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  useEffect(() => {
    void load();
    api<typeof currencies>("currencies")
      .then(setCurrencies)
      .catch((e: Error) => msg.error(e.message));
  }, []);
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
      form.setFieldsValue(result);
      await load();
    } catch (e) {
      msg.error((e as Error).message || "请检查必填项");
    } finally {
      setBusy(false);
    }
  }
  return (
    <Card
      title="产品管理"
      extra={
        can("product.create") && (
          <Button type="primary" onClick={() => void edit()}>
            新增产品
          </Button>
        )
      }
    >
      {context}
      <Input.Search
        placeholder="搜索产品名称、CAS 或分类"
        onSearch={(value) => {
          setQuery(value);
          void load(1, data.pageSize, value);
        }}
      />
      <Table<ProductSummary>
        loading={busy}
        rowKey="id"
        dataSource={data.items}
        pagination={{
          current: data.page,
          pageSize: data.pageSize,
          total: data.total,
          showSizeChanger: true,
          onChange: load,
        }}
        columns={[
          { title: "名称", dataIndex: "name" },
          {
            title: "价格",
            render: (_, row) => row.fromPrice?.displayPrice || "未定价",
          },
          {
            title: "状态",
            render: (_, row) => (row.status ? labels[row.status] : ""),
          },
          {
            title: "操作",
            render: (_, row) => (
              <Space wrap>
                {(can("product.update") || can("product.preview")) && (
                  <Button onClick={() => void edit(row.id)}>查看 / 编辑</Button>
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
                  <Button
                    onClick={async () => {
                      try {
                        await api(`products/${row.id}/status`, "PATCH", {
                          status:
                            row.status === "published"
                              ? "offline"
                              : "published",
                          expectedVersion: row.version,
                        });
                        await load();
                      } catch (e) {
                        msg.error((e as Error).message);
                      }
                    }}
                  >
                    {row.status === "published" ? "下架" : "上架"}
                  </Button>
                )}
                {can("product.delete") && (
                  <Popconfirm
                    title="确认删除产品？"
                    onConfirm={async () => {
                      try {
                        await api(`products/${row.id}`, "DELETE", {
                          expectedVersion: row.version,
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
        title={editing ? "编辑产品" : "新增产品"}
        open={editing !== undefined}
        onCancel={() => setEditing(undefined)}
        width={1100}
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
            <Button onClick={() => setEditing(undefined)}>关闭</Button>
          </Space>
        }
        destroyOnHidden
      >
        <Form form={form} layout="vertical" preserve={false}>
          <Form.Item
            name="name"
            label="产品名称（英文）"
            rules={[{ required: true, max: 200 }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="slug"
            label="官网路径（小写字母、数字及连字符）"
            rules={[{ required: true, pattern: /^[a-z0-9]+(?:-[a-z0-9]+)*$/ }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="summary" label="简介（英文）">
            <Input.TextArea />
          </Form.Item>
          <Space wrap>
            {(
              [
                "category",
                "aliases",
                "cas",
                "formula",
                "molecularWeight",
                "sequence",
                "purity",
                "storage",
                "solubility",
              ] as const
            ).map((key) => (
              <Form.Item
                key={key}
                name={key}
                label={
                  {
                    category: "分类",
                    aliases: "别名",
                    cas: "CAS",
                    formula: "分子式",
                    molecularWeight: "分子量",
                    sequence: "序列",
                    purity: "纯度",
                    storage: "存储条件",
                    solubility: "溶解性",
                  }[key]
                }
              >
                <Input />
              </Form.Item>
            ))}
          </Space>
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
                title: "价格（十进制字符串）",
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
                        onClick={() => setPriceEdit({ product: editing, sku })}
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
          <Form.Item name="detail" label="产品详情（英文，媒体从库选择）">
            <RichTextEditor />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select
              disabled={!can("product.publish")}
              options={Object.entries(labels).map(([value, label]) => ({
                value,
                label,
              }))}
            />
          </Form.Item>
          <Form.Item name="seoTitle" label="SEO 标题">
            <Input />
          </Form.Item>
          <Form.Item name="seoDescription" label="SEO 描述">
            <Input.TextArea />
          </Form.Item>
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
              form.setFieldsValue(result);
              await load();
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
    </Card>
  );
}
