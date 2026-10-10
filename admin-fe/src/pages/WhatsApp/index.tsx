import { useEffect, useState } from "react";
import { useModel } from "@umijs/max";
import {
  Alert,
  Button,
  Card,
  Form,
  Image,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Spin,
  Switch,
  Table,
  Tag,
  message,
} from "antd";
import MediaSelect from "../../components/MediaSelect";
import { api, assetURL, APIException } from "../../services/cms";
import { allows } from "../../services/auth";
import type {
  WhatsAppAccount,
  WhatsAppGroup,
  WhatsAppGroupInput,
  WhatsAppGroupPage,
} from "../../services/contracts";

function QRPicker({
  value,
  onChange,
}: {
  value?: string;
  onChange?: (v: string) => void;
}) {
  const [url, setURL] = useState("");
  useEffect(() => {
    let active = true;
    setURL("");
    if (value)
      api<{ url: string }>(`media/${value}`)
        .then((m) => {
          if (active) setURL(m.url);
        })
        .catch(() => {});
    return () => {
      active = false;
    };
  }, [value]);
  return (
    <Space direction="vertical">
      <MediaSelect value={value} onChange={onChange} />
      {url && <Image width={140} src={assetURL(url)} />}
    </Space>
  );
}
const regionNames = new Intl.DisplayNames(["zh-CN"], { type: "region" });
const region = (code: string) => `${regionNames.of(code) || code} (${code})`;
const empty: WhatsAppGroupPage = {
  items: [],
  total: 0,
  page: 1,
  pageSize: 20,
  defaultGroupId: "",
  defaultGroupName: "",
  routingUpdatedAt: "",
  geoipAvailable: false,
};
export default function WhatsAppPage() {
  const { initialState } = useModel("@@initialState");
  const can = (p: string) => allows(initialState?.identity, p);
  const [data, setData] = useState(empty);
  const [countries, setCountries] = useState<{ code: string; name: string }[]>(
    [],
  );
  const [details, setDetails] = useState<Record<string, WhatsAppGroup>>({});
  const [expanded, setExpanded] = useState<React.Key[]>([]);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<WhatsAppGroup>();
  const [conflict, setConflict] = useState(false);
  const [form] = Form.useForm<WhatsAppGroupInput>();
  const [msg, context] = message.useMessage();
  async function load(page = data.page, q = query) {
    setLoading(true);
    try {
      const next = await api<WhatsAppGroupPage>(
        `whatsapp-groups?page=${page}&pageSize=20&q=${encodeURIComponent(q)}`,
      );
      setData(next);
      setDetails({});
      setExpanded([]);
    } catch (e) {
      msg.error((e as Error).message);
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    void load(1);
    api<{ items: { code: string; name: string }[] }>("whatsapp-countries")
      .then((v) => setCountries(v.items))
      .catch((e) => msg.error(e.message));
  }, []);
  async function expand(row: WhatsAppGroup) {
    try {
      const current = await api<WhatsAppGroup>(`whatsapp-groups/${row.id}`);
      setDetails((prev) => ({ ...prev, [row.id]: current }));
      setData((prev) => ({
        ...prev,
        items: prev.items.map((g) =>
          g.id === current.id
            ? { ...current, isDefault: current.id === prev.defaultGroupId }
            : g,
        ),
      }));
    } catch (e) {
      msg.error((e as Error).message);
      setExpanded((keys) => keys.filter((k) => k !== row.id));
    }
  }
  async function edit(row?: WhatsAppGroup) {
    try {
      const current = row
        ? await api<WhatsAppGroup>(`whatsapp-groups/${row.id}`)
        : undefined;
      setEditing(current);
      setConflict(false);
      form.resetFields();
      form.setFieldsValue(
        current
          ? {
              name: current.name,
              countryCodes: current.countryCodes,
              enabled: current.enabled,
              accounts: current.accounts?.map((a) => ({
                id: a.id,
                name: a.name,
                phoneNumber: a.phoneNumber,
                qrCodeMediaId: a.qrCodeMediaId,
                enabled: a.enabled,
              })),
            }
          : {
              name: "",
              countryCodes: [],
              enabled: true,
              accounts: [
                { name: "", phoneNumber: "", qrCodeMediaId: "", enabled: true },
              ],
            },
      );
      setOpen(true);
    } catch (e) {
      msg.error((e as Error).message);
    }
  }
  async function fail(e: unknown) {
    msg.error((e as Error).message);
    if (e instanceof APIException && e.code === "UPDATE_CONFLICT") await load();
  }
  async function toggle(
    row: WhatsAppGroup,
    enabled: boolean,
    account?: WhatsAppAccount,
  ) {
    setBusy(true);
    try {
      const current = await api<WhatsAppGroup>(
        `whatsapp-groups/${row.id}${account ? `/accounts/${account.id}` : ""}/enabled`,
        "PATCH",
        { enabled, expectedUpdatedAt: row.updatedAt },
      );
      setDetails((prev) => ({ ...prev, [row.id]: current }));
      setData((prev) => ({
        ...prev,
        items: prev.items.map((g) =>
          g.id === current.id
            ? { ...current, isDefault: current.id === prev.defaultGroupId }
            : g,
        ),
      }));
      msg.success("状态已更新");
    } catch (e) {
      await fail(e);
    } finally {
      setBusy(false);
    }
  }
  async function save(values: WhatsAppGroupInput) {
    setBusy(true);
    try {
      await api(
        `whatsapp-groups${editing ? `/${editing.id}` : ""}`,
        editing ? "PUT" : "POST",
        {
          name: values.name,
          countryCodes: values.countryCodes,
          enabled: values.enabled,
          accounts: values.accounts,
          ...(editing ? { expectedUpdatedAt: editing.updatedAt } : {}),
        },
      );
      setOpen(false);
      msg.success("配置已保存");
      await load();
    } catch (e) {
      if (e instanceof APIException && e.code === "UPDATE_CONFLICT")
        setConflict(true);
      await fail(e);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Card
      title="WhatsApp 客服配置"
      extra={
        can("whatsapp.create") && (
          <Button type="primary" onClick={() => void edit()}>
            新增配置
          </Button>
        )
      }
    >
      {context}
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message={`默认配置：${data.defaultGroupName || "尚未设置"}。${data.geoipAvailable ? "已启用 IP 国家识别。" : "尚未配置国家数据库，访客使用默认配置。"}`}
      />
      <Input.Search
        placeholder="搜索配置名称或国家代码"
        style={{ maxWidth: 360, marginBottom: 16 }}
        onSearch={(q) => {
          setQuery(q);
          void load(1, q);
        }}
      />
      <Table<WhatsAppGroup>
        rowKey="id"
        loading={loading}
        dataSource={data.items}
        scroll={{ x: 1000 }}
        pagination={{
          current: data.page,
          pageSize: 20,
          total: data.total,
          onChange: (p) => void load(p),
        }}
        expandable={{
          expandedRowKeys: expanded,
          onExpand: (yes, row) => {
            setExpanded((keys) =>
              yes ? [...keys, row.id] : keys.filter((k) => k !== row.id),
            );
            if (yes) void expand(row);
          },
          expandedRowRender: (row) =>
            details[row.id] ? (
              <Table<WhatsAppAccount>
                rowKey="id"
                pagination={false}
                dataSource={details[row.id].accounts}
                columns={[
                  { title: "客服名称", dataIndex: "name" },
                  { title: "国际号码", dataIndex: "phoneNumber" },
                  { title: "累计分配", dataIndex: "allocationCount" },
                  {
                    title: "启用状态",
                    render: (_, a) => (
                      <Switch
                        checked={a.enabled}
                        disabled={!can("whatsapp.update") || busy}
                        checkedChildren="启用"
                        unCheckedChildren="停用"
                        onChange={(yes) => void toggle(details[row.id], yes, a)}
                      />
                    ),
                  },
                ]}
              />
            ) : (
              <Spin />
            ),
        }}
        columns={[
          { title: "配置名称", dataIndex: "name" },
          {
            title: "国家／地区",
            render: (_, g) => g.countryCodes.map(region).join("、"),
          },
          {
            title: "状态",
            render: (_, g) => (
              <Space>
                <Tag color={g.enabled ? "green" : "default"}>
                  {g.enabled ? "启用" : "停用"}
                </Tag>
                {g.pendingSetup && <Tag color="orange">待配置联系方式</Tag>}
              </Space>
            ),
          },
          {
            title: "可用／全部账号",
            render: (_, g) => `${g.availableAccountCount} / ${g.accountCount}`,
          },
          {
            title: "默认",
            render: (_, g) =>
              g.isDefault ? <Tag color="blue">默认配置</Tag> : "—",
          },
          {
            title: "操作",
            render: (_, g) => (
              <Space wrap>
                {can("whatsapp.update") && (
                  <>
                    <Button disabled={busy} onClick={() => void edit(g)}>
                      编辑
                    </Button>
                    <Button
                      disabled={busy}
                      onClick={() => void toggle(g, !g.enabled)}
                    >
                      {g.enabled ? "停用" : "启用"}
                    </Button>
                  </>
                )}
                {can("whatsapp.default") && (
                  <Popconfirm
                    title={`将「${g.name}」设为唯一默认配置？`}
                    description={
                      data.defaultGroupName
                        ? `当前默认「${data.defaultGroupName}」将被替换，未匹配地域访客将使用新配置。`
                        : "未匹配地域访客将使用此配置的 WhatsApp 账号。"
                    }
                    okText="确认设置"
                    cancelText="取消"
                    disabled={busy || g.isDefault}
                    onConfirm={async () => {
                      setBusy(true);
                      try {
                        await api("whatsapp-default", "PUT", {
                          groupId: g.id,
                          expectedUpdatedAt: data.routingUpdatedAt,
                          expectedGroupUpdatedAt: g.updatedAt,
                        });
                        msg.success("已设为默认");
                        await load();
                      } catch (e) {
                        await fail(e);
                      } finally {
                        setBusy(false);
                      }
                    }}
                  >
                    <Button disabled={busy || g.isDefault}>
                      {g.isDefault ? "当前默认" : "设为默认"}
                    </Button>
                  </Popconfirm>
                )}
                {can("whatsapp.delete") && (
                  <Popconfirm
                    title="删除这条配置及其账号？"
                    onConfirm={async () => {
                      setBusy(true);
                      try {
                        await api(`whatsapp-groups/${g.id}`, "DELETE", {
                          expectedUpdatedAt: g.updatedAt,
                        });
                        msg.success("已删除");
                        await load();
                      } catch (e) {
                        await fail(e);
                      } finally {
                        setBusy(false);
                      }
                    }}
                  >
                    <Button danger disabled={busy || g.isDefault}>
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
        open={open}
        title={editing ? "编辑客服配置" : "新增客服配置"}
        width={850}
        onCancel={() => {
          if (!busy) setOpen(false);
        }}
        footer={null}
        destroyOnHidden
      >
        {conflict && (
          <Alert
            type="warning"
            message="配置已被其他人修改，当前输入已保留。请记录需要保留的内容，再重新载入核对后保存。"
            action={
              <Button disabled={busy} onClick={() => void edit(editing)}>
                重新载入
              </Button>
            }
            style={{ marginBottom: 16 }}
          />
        )}
        <Form
          form={form}
          layout="vertical"
          onFinish={(values) => void save(values)}
          disabled={busy}
        >
          <Form.Item
            name="name"
            label="配置名称"
            rules={[{ required: true, max: 200 }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="countryCodes"
            label="国家／地区"
            rules={[{ required: true, type: "array", min: 1 }]}
          >
            <Select
              mode="multiple"
              showSearch
              optionFilterProp="label"
              options={countries.map((c) => ({
                value: c.code,
                label: region(c.code),
              }))}
            />
          </Form.Item>
          <Form.Item name="enabled" label="配置启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.List name="accounts">
            {(fields, { add, remove }) => (
              <>
                {fields.map((field) => (
                  <Card
                    key={field.key}
                    size="small"
                    title={`账号 ${field.name + 1}`}
                    style={{ marginBottom: 12 }}
                    extra={
                      <Button
                        danger
                        disabled={busy}
                        onClick={() => remove(field.name)}
                      >
                        移除
                      </Button>
                    }
                  >
                    <Form.Item name={[field.name, "id"]} hidden>
                      <Input />
                    </Form.Item>
                    <Form.Item
                      name={[field.name, "name"]}
                      label="客服名称"
                      rules={[{ required: true, max: 100 }]}
                    >
                      <Input />
                    </Form.Item>
                    <Form.Item
                      name={[field.name, "phoneNumber"]}
                      label="国际号码（区号加号码，仅数字）"
                      rules={[
                        { required: true, pattern: /^[1-9][0-9]{6,14}$/ },
                      ]}
                    >
                      <Input placeholder="不包含加号、空格或横线" />
                    </Form.Item>
                    <Form.Item
                      name={[field.name, "qrCodeMediaId"]}
                      label="WhatsApp 账号二维码"
                      rules={[
                        { required: true, message: "请选择账号二维码图片" },
                      ]}
                    >
                      <QRPicker />
                    </Form.Item>
                    <Form.Item
                      name={[field.name, "enabled"]}
                      label="账号启用"
                      valuePropName="checked"
                    >
                      <Switch />
                    </Form.Item>
                  </Card>
                ))}
                <Button
                  disabled={busy || fields.length >= 100}
                  onClick={() =>
                    add({
                      name: "",
                      phoneNumber: "",
                      qrCodeMediaId: "",
                      enabled: true,
                    })
                  }
                >
                  添加账号
                </Button>
              </>
            )}
          </Form.List>
          <Space style={{ marginTop: 20 }}>
            <Button
              type="primary"
              htmlType="submit"
              loading={busy}
              disabled={conflict}
            >
              保存配置
            </Button>
            <Button onClick={() => setOpen(false)} disabled={busy}>
              取消
            </Button>
          </Space>
        </Form>
      </Modal>
    </Card>
  );
}
