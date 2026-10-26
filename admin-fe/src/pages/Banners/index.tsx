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
  Select,
  Space,
  Table,
  Tag,
  message,
} from "antd";
import { api, type Banner } from "../../services/cms";
import { allows } from "../../services/auth";
import type { Page } from "../../services/contracts";
import MediaSelect from "../../components/MediaSelect";
const localDate = (s: string | null) => {
  if (!s) return "";
  const date = new Date(s);
  return new Date(date.getTime() + 8 * 3600000).toISOString().slice(0, 16);
};
export default function BannersPage() {
  const { initialState } = useModel("@@initialState");
  const can = (p: string) => allows(initialState?.identity, p);
  const [slots, setSlots] = useState<{ code: string; name: string }[]>([]);
  const [slot, setSlot] = useState("");
  const [data, setData] = useState<Page<Banner>>({
    items: [],
    total: 0,
    page: 1,
    pageSize: 20,
  });
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<Banner>();
  const [busy, setBusy] = useState(false);
  const [form] = Form.useForm();
  const [msg, context] = message.useMessage();
  const isHome = Form.useWatch("slotCode", form) === "home_hero";
  const [productQuery, setProductQuery] = useState("");
  const [products, setProducts] = useState<{ id: string; name: string }[]>([]);
  const [searching, setSearching] = useState(false);
  useEffect(() => {
    if (!open || !isHome) return;
    let active = true;
    setSearching(true);
    const timer = window.setTimeout(() => {
      api<{ items: { id: string; name: string }[] }>(`banner-products?q=${encodeURIComponent(productQuery)}`)
        .then((v) => { if (active) setProducts(v.items); })
        .catch((e) => { if (active) msg.error(e.message); })
        .finally(() => { if (active) setSearching(false); });
    }, 250);
    return () => { active = false; window.clearTimeout(timer); };
  }, [open, isHome, productQuery]);
  async function load(page = data.page, code = slot) {
    try {
      setData(
        await api(
          `banners?page=${page}&pageSize=20&slot=${encodeURIComponent(code)}`,
        ),
      );
    } catch (e) {
      msg.error((e as Error).message);
    }
  }
  useEffect(() => {
    void load(1);
    api<{ items: { code: string; name: string }[] }>("banner-slots")
      .then((v) => setSlots(v.items))
      .catch((e) => msg.error(e.message));
  }, []);
  async function edit(row?: Banner) {
    try {
      const b = row ? await api<Banner>(`banners/${row.id}`) : undefined;
      setEditing(b);
      setProductQuery("");
      setProducts([]);
      form.resetFields();
      form.setFieldsValue(
        b
          ? {
              ...b,
              startsAt: localDate(b.startsAt),
              endsAt: localDate(b.endsAt),
            }
          : {
              slotCode: slot || slots[0]?.code,
              productId: "",
              title: "",
              subtitle: "",
              imageMediaId: "",
              mobileImageMediaId: "",
              ctaText: "",
              link: "",
              sort: 0,
              startsAt: "",
              endsAt: "",
            },
      );
      setOpen(true);
    } catch (e) {
      msg.error((e as Error).message);
    }
  }
  return (
    <Card
      title="Banner管理"
      extra={
        can("banner.create") && (
          <Button type="primary" onClick={() => void edit()}>
            新增 Banner
          </Button>
        )
      }
    >
      {context}
      <Select
        value={slot}
        style={{ width: 240, marginBottom: 20 }}
        options={[
          { value: "", label: "全部位置" },
          ...slots.map((s) => ({ value: s.code, label: s.name })),
        ]}
        onChange={(s) => {
          setSlot(s);
          void load(1, s);
        }}
      />
      <Table<Banner>
        rowKey="id"
        dataSource={data.items}
        pagination={{
          current: data.page,
          pageSize: 20,
          total: data.total,
          onChange: (p) => void load(p),
        }}
        columns={[
          { title: "名称", dataIndex: "title" },
          { title: "关联商品", render: (_, b) => b.slotCode === "home_hero" ? (b.productName || (b.productId ? `商品 #${b.productId}（不可用）` : "未关联")) : "—" },
          { title: "位置", dataIndex: "slotCode" },
          { title: "排序", dataIndex: "sort" },
          {
            title: "状态",
            render: (_, b) => <Tag>{b.enabled ? "启用" : "停用"}</Tag>,
          },
          {
            title: "排期（北京时间）",
            render: (_, b) =>
              b.slotCode === "home_hero" ? "—" : `${localDate(b.startsAt) || "立即"} 至 ${localDate(b.endsAt) || "不限"}`,
          },
          {
            title: "操作",
            render: (_, b) => (
              <Space>
                {can("banner.update") && (
                  <Button onClick={() => void edit(b)}>编辑</Button>
                )}
                {can("banner.publish") && (
                  <Button
                    onClick={async () => {
                      try {
                        await api(`banners/${b.id}/enabled`, "PATCH", {
                          enabled: !b.enabled,
                          expectedVersion: b.version,
                        });
                        await load();
                      } catch (e) {
                        msg.error((e as Error).message);
                      }
                    }}
                  >
                    {b.enabled ? "停用" : "启用"}
                  </Button>
                )}
                {can("banner.delete") && (
                  <Popconfirm
                    title="确认删除投放？"
                    onConfirm={async () => {
                      try {
                        await api(`banners/${b.id}`, "DELETE", {
                          expectedVersion: b.version,
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
        title={editing ? "编辑 Banner" : "新增 Banner"}
        open={open}
        width={760}
        destroyOnHidden
        confirmLoading={busy}
        onCancel={() => setOpen(false)}
        onOk={() => form.submit()}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={async (v) => {
            setBusy(true);
            try {
              await api(
                editing ? `banners/${editing.id}` : "banners",
                editing ? "PUT" : "POST",
                {
                  ...v,
                  productId: v.slotCode === "home_hero" ? (v.productId || "") : "",
                  subtitle: v.slotCode === "home_hero" ? "" : v.subtitle,
                  ctaText: v.slotCode === "home_hero" ? "" : v.ctaText,
                  link: v.slotCode === "home_hero" ? "" : v.link,
                  startsAt: v.slotCode !== "home_hero" && v.startsAt
                    ? new Date(v.startsAt + ":00+08:00").toISOString()
                    : null,
                  endsAt: v.slotCode !== "home_hero" && v.endsAt
                    ? new Date(v.endsAt + ":00+08:00").toISOString()
                    : null,
                  enabled: editing?.enabled || false,
                  expectedVersion: editing?.version || 0,
                },
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
          <Form.Item name="slotCode" label="位置" rules={[{ required: true }]}>
            <Select
              options={slots.map((s) => ({ value: s.code, label: s.name }))}
            />
          </Form.Item>
          <Form.Item
            name="title"
            label={isHome ? "后台名称（不展示给访客）" : "英文标题"}
            rules={[{ required: true, max: 200 }]}
          >
            <Input />
          </Form.Item>
          <Form.Item hidden={isHome} name="subtitle" label="英文副标题">
            <Input.TextArea maxLength={2000} />
          </Form.Item>
          <Form.Item
            name="imageMediaId"
            label="桌面图片"
            extra={isHome ? "建议 1920×720，图片左下方预留按钮区域；所有桌面图保持相同比例。" : undefined}
            rules={[{ required: true }]}
          >
            <MediaSelect />
          </Form.Item>
          <Form.Item
            name="mobileImageMediaId"
            label="手机图片（留空使用桌面图片）"
            extra={isHome ? "建议 800×1000，底部预留按钮和圆点区域。" : undefined}
          >
            <MediaSelect />
          </Form.Item>
          {isHome && <>
            <Form.Item name="productId" label="关联商品（选填）" extra="选择商品后显示按钮；未关联或商品不可访问时隐藏按钮。">
              <Select showSearch allowClear filterOption={false} onSearch={setProductQuery} loading={searching} placeholder="输入商品名称搜索"
                options={[
                  ...(editing?.productId && !products.some((p) => p.id === editing.productId) ? [{ value: editing.productId, label: editing.productName || `商品 #${editing.productId}（不可用）` }] : []),
                  ...products.map((p) => ({ value: p.id, label: p.name })),
                ]} />
            </Form.Item>
            <Form.Item label="按钮文案"><Input value="Request In-Vitro Quotation ↗" disabled /></Form.Item>
          </>}
          <Form.Item hidden={isHome} name="ctaText" label="英文按钮文字">
            <Input maxLength={100} />
          </Form.Item>
          <Form.Item hidden={isHome} name="link" label="链接（站内路径或 HTTPS）">
            <Input maxLength={2000} />
          </Form.Item>
          <Form.Item name="sort" label="排序（较小优先）">
            <InputNumber precision={0} />
          </Form.Item>
          <Form.Item hidden={isHome} name="startsAt" label="开始时间（Asia/Shanghai）">
            <Input type="datetime-local" />
          </Form.Item>
          <Form.Item hidden={isHome} name="endsAt" label="结束时间（Asia/Shanghai）">
            <Input type="datetime-local" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
