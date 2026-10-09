import { useEffect, useRef, useState } from "react";
import { Button, Card, Form, Image, Input, Modal, Popconfirm, Select, Space, Table, Tag, message } from "antd";
import { api, assetURL, type Product } from "../../services/cms";
import MediaSelect from "../../components/MediaSelect";

const statusOptions = [
 { label: "草稿", value: "draft" },
 { label: "已上架", value: "published" },
 { label: "已下架", value: "offline" },
];
const fields: [string, string, number][] = [
 ["category", "分类", 200], ["aliases", "别名", 5000], ["cas", "CAS 号", 200],
 ["formula", "分子式", 200], ["molecularWeight", "分子量", 200], ["sequence", "序列", 5000],
 ["purity", "纯度", 200], ["specifications", "规格", 5000], ["storage", "储存说明", 5000],
 ["solubility", "溶解说明", 5000], ["summary", "摘要", 5000],
 ["description", "详情（纯文本）", 100000], ["seoTitle", "SEO 标题", 200], ["seoDescription", "SEO 描述", 5000],
];

export default function ProductsPage() {
 const [rows, setRows] = useState<Product[]>([]);
 const [open, setOpen] = useState(false);
 const [busy, setBusy] = useState(false);
 const [loading, setLoading] = useState(false);
 const [search, setSearch] = useState("");
 const [editing, setEditing] = useState<Product>();
 const [form] = Form.useForm();
 const [msg, context] = message.useMessage();
 const requestID = useRef(0);
 async function load(q = search) {
  const id = ++requestID.current;
  setLoading(true);
  try {
   const result = await api<Product[]>(`products?q=${encodeURIComponent(q)}`);
   if (id === requestID.current) setRows(result);
  } catch (e) { if (id === requestID.current) msg.error((e as Error).message); }
  finally { if (id === requestID.current) setLoading(false); }
 }
 useEffect(() => { void load(""); }, []);
 function edit(p?: Product) {
  setEditing(p);
  form.resetFields();
  form.setFieldsValue(p ?? { name: "", slug: "", status: "draft", images: [], ...Object.fromEntries(fields.map(([name]) => [name, ""])) });
  setOpen(true);
 }
 return <Card title="商品管理" extra={<Button type="primary" onClick={() => edit()}>新增商品</Button>}>
  {context}
  <Input.Search placeholder="搜索名称、别名、CAS、序列或分类" maxLength={200} style={{ maxWidth: 480, marginBottom: 20 }} allowClear onSearch={q => { setSearch(q); void load(q); }} />
  <Table<Product> rowKey="id" dataSource={rows} loading={loading} columns={[
   { title: "图片", render: (_, r) => r.images?.[0] ? <Image width={60} height={60} style={{ objectFit: "contain" }} src={assetURL(r.images[0])} /> : "—" },
   { title: "名称", dataIndex: "name" }, { title: "分类", dataIndex: "category" }, { title: "CAS", dataIndex: "cas" },
   { title: "状态", render: (_, r) => <Tag color={r.status === "published" ? "green" : "default"}>{statusOptions.find(s => s.value === r.status)?.label}</Tag> },
   { title: "操作", render: (_, r) => <Space>
    <Button onClick={() => edit(r)}>编辑</Button>
    <Button disabled={busy} onClick={async () => {
     setBusy(true);
     try { await api("products", "PUT", { ...r, status: r.status === "published" ? "offline" : "published" }); await load(); msg.success("状态已更新"); }
     catch (e) { msg.error((e as Error).message); } finally { setBusy(false); }
    }}>{r.status === "published" ? "下架" : "上架"}</Button>
    <Popconfirm title="确定删除商品？" onConfirm={async () => {
     try { await api(`products/${r.id}`, "DELETE"); await load(); msg.success("已删除"); }
     catch (e) { msg.error((e as Error).message); }
    }}><Button danger>删除</Button></Popconfirm>
   </Space> },
  ]} />
  <Modal title={editing ? "编辑商品" : "新增商品"} open={open} width={850} onCancel={() => setOpen(false)} confirmLoading={busy} onOk={() => form.submit()} destroyOnHidden>
   <Form form={form} layout="vertical" onFinish={async v => {
    setBusy(true);
    try {
     await api("products", editing ? "PUT" : "POST", { ...v, id: editing?.id ?? 0, images: v.images ?? [] });
     setOpen(false); await load(); msg.success("已保存");
    } catch (e) { msg.error((e as Error).message); } finally { setBusy(false); }
   }}>
    <Form.Item name="name" label="英文名称" rules={[{ required: true, whitespace: true, max: 200 }]}><Input maxLength={200} /></Form.Item>
    <Form.Item name="slug" label="Slug（小写英文、数字、连字符）" rules={[{ required: true, max: 120, pattern: /^[a-z0-9]+(?:-[a-z0-9]+)*$/ }]}><Input maxLength={120} /></Form.Item>
    <Form.Item name="status" label="状态" rules={[{ required: true }]}><Select options={statusOptions} /></Form.Item>
    <Form.List name="images" rules={[{ validator: async (_, images) => { if ((images?.length ?? 0) > 20) throw new Error("最多 20 张图片"); } }]}>
     {(items, { add, remove }, { errors }) => <div style={{ marginBottom: 20 }}>
      <p>商品图片（第一张为主图，最多 20 张）</p>
      {items.map(item => <Space key={item.key} align="start" style={{ display: "flex", marginBottom: 8 }}>
       <Form.Item name={item.name} rules={[{ required: true, message: "请选择图片" }]} style={{ width: 520, marginBottom: 0 }}><MediaSelect /></Form.Item>
       <Button onClick={() => remove(item.name)}>移除</Button>
      </Space>)}
      <Button onClick={() => add("")} disabled={items.length >= 20}>添加图片</Button>
      <Form.ErrorList errors={errors} />
     </div>}
    </Form.List>
    {fields.map(([name, label, max]) => <Form.Item key={name} name={name} label={label} rules={[{ max }]}>
     {max > 200 ? <Input.TextArea rows={name === "description" ? 8 : 2} maxLength={max} /> : <Input maxLength={max} />}
    </Form.Item>)}
   </Form>
  </Modal>
 </Card>;
}
