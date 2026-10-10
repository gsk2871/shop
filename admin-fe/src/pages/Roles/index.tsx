import { useEffect, useState } from "react";
import { useModel } from "@umijs/max";
import {
  Button,
  Card,
  Form,
  Input,
  Modal,
  Popconfirm,
  Space,
  Switch,
  Table,
  Tree,
  message,
} from "antd";
import { api } from "../../services/cms";
import { allows, type Role } from "../../services/auth";
import type { Page, PermissionNode } from "../../services/contracts";
function tree(
  nodes: PermissionNode[],
): { key: string; title: string; children: ReturnType<typeof tree> }[] {
  return nodes.map((node) => ({
    key: node.permissionCode || node.code,
    title: node.name,
    children: tree(node.children),
  }));
}
function permissions(nodes: PermissionNode[]): string[] {
  return nodes.flatMap((node) => [
    ...(node.permissionCode ? [node.permissionCode] : []),
    ...permissions(node.children),
  ]);
}
export default function Roles() {
  const { initialState } = useModel("@@initialState");
  const can = (p: string) => allows(initialState?.identity, p);
  const [data, setData] = useState<Page<Role>>({
    items: [],
    total: 0,
    page: 1,
    pageSize: 20,
  });
  const [editing, setEditing] = useState<Role | null>();
  const [granting, setGranting] = useState<Role>();
  const [nodes, setNodes] = useState<PermissionNode[]>([]);
  const [checked, setChecked] = useState<string[]>([]);
  const [form] = Form.useForm();
  const [busy, setBusy] = useState(false);
  const [msg, context] = message.useMessage();
  async function load(page = data.page, pageSize = data.pageSize) {
    try {
      setData(await api(`roles?page=${page}&pageSize=${pageSize}`));
    } catch (e) {
      msg.error((e as Error).message);
    }
  }
  async function act(path: string, method: string, body?: unknown) {
    setBusy(true);
    try {
      await api(path, method, body);
      await load();
      msg.success("操作成功");
      return true;
    } catch (e) {
      msg.error((e as Error).message);
      return false;
    } finally {
      setBusy(false);
    }
  }
  useEffect(() => {
    void load();
    api<PermissionNode[]>("permissions")
      .then(setNodes)
      .catch((e: Error) => msg.error(e.message));
  }, []);
  function edit(role: Role | null) {
    form.resetFields();
    form.setFieldsValue(role || { enabled: true });
    setEditing(role);
  }
  return (
    <Card
      title="角色管理"
      extra={
        can("role.create") && (
          <Button type="primary" onClick={() => edit(null)}>
            新增角色
          </Button>
        )
      }
    >
      {context}
      <Table<Role>
        rowKey="id"
        dataSource={data.items}
        pagination={{
          current: data.page,
          total: data.total,
          pageSize: data.pageSize,
          onChange: load,
        }}
        columns={[
          { title: "代码", dataIndex: "code" },
          { title: "名称", dataIndex: "name" },
          { title: "启用", render: (_, role) => (role.enabled ? "是" : "否") },
          {
            title: "操作",
            render: (_, role) => (
              <Space>
                {can("role.update") && (
                  <Button onClick={() => edit(role)}>编辑</Button>
                )}
                {can("role.grant") && !role.isSuper && (
                  <Button
                    onClick={async () => {
                      try {
                        const result = await api<{ permissionCodes: string[] }>(
                          `roles/${role.id}/permissions`,
                        );
                        setChecked(result.permissionCodes);
                        setGranting(role);
                      } catch (e) {
                        msg.error((e as Error).message);
                      }
                    }}
                  >
                    授权
                  </Button>
                )}
                {can("role.delete") && !role.isSuper && (
                  <Popconfirm
                    title="确认删除角色？"
                    onConfirm={() => act(`roles/${role.id}`, "DELETE")}
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
        title={editing ? "编辑角色" : "新增角色"}
        open={editing !== undefined}
        onCancel={() => setEditing(undefined)}
        onOk={() => form.submit()}
        confirmLoading={busy}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={async (values) => {
            if (
              await act(
                editing ? `roles/${editing.id}` : "roles",
                editing ? "PUT" : "POST",
                values,
              )
            )
              setEditing(undefined);
          }}
        >
          <Form.Item
            name="code"
            label="角色代码"
            rules={[{ required: true, pattern: /^[a-z][a-z0-9_]{0,99}$/ }]}
          >
            <Input disabled={!!editing} />
          </Form.Item>
          <Form.Item
            name="name"
            label="角色名称"
            rules={[{ required: true, max: 100 }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch disabled={editing?.isSuper} />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title={`角色授权 · ${granting?.name || ""}`}
        open={!!granting}
        onCancel={() => setGranting(undefined)}
        confirmLoading={busy}
        onOk={async () => {
          if (
            granting &&
            (await act(`roles/${granting.id}/permissions`, "PUT", {
              permissionCodes: checked.filter((code) =>
                permissions(nodes).includes(code),
              ),
            }))
          )
            setGranting(undefined);
        }}
      >
        <p>授权变更将撤销此角色用户的已有会话，下一次请求即生效。</p>
        <Tree
          checkable
          defaultExpandAll
          treeData={tree(nodes)}
          checkedKeys={checked}
          onCheck={(keys) =>
            setChecked((Array.isArray(keys) ? keys : keys.checked).map(String))
          }
        />
      </Modal>
    </Card>
  );
}
