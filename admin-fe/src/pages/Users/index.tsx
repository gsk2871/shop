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
import { api } from "../../services/cms";
import { allows, allRoles, type User, type Role } from "../../services/auth";
import type { Page } from "../../services/contracts";
export default function Users() {
  const { initialState } = useModel("@@initialState");
  const can = (p: string) => allows(initialState?.identity, p);
  const [data, setData] = useState<Page<User>>({
    items: [],
    total: 0,
    page: 1,
    pageSize: 20,
  });
  const [roles, setRoles] = useState<Role[]>([]);
  const [editing, setEditing] = useState<User | null>();
  const [reset, setReset] = useState<{ user: User; action: string }>();
  const [busy, setBusy] = useState(false);
  const [form] = Form.useForm();
  const [msg, context] = message.useMessage();
  async function load(page = data.page, pageSize = data.pageSize) {
    setBusy(true);
    try {
      setData(await api(`users?page=${page}&pageSize=${pageSize}`));
    } catch (e) {
      msg.error((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  async function act(path: string, method: string, body?: unknown) {
    setBusy(true);
    try {
      await api(path, method, body);
      msg.success("操作成功");
      await load();
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
    allRoles()
      .then(setRoles)
      .catch(() => setRoles([]));
  }, []);
  function edit(user: User | null) {
    setEditing(user);
    form.resetFields();
    form.setFieldsValue(user || { enabled: true });
  }
  return (
    <Card
      title="用户管理"
      extra={
        can("user.create") && (
          <Button type="primary" onClick={() => edit(null)}>
            新增用户
          </Button>
        )
      }
    >
      {context}
      <Table<User>
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
          { title: "账号", dataIndex: "username" },
          { title: "名称", dataIndex: "displayName" },
          {
            title: "角色",
            render: (_, user) =>
              roles.find((role) => role.id === user.roleId)?.name ||
              user.roleId,
          },
          {
            title: "启用",
            render: (_, user) => (
              <Switch
                checked={user.enabled}
                disabled={!can("user.disable")}
                onChange={(enabled) =>
                  void act(`users/${user.id}/status`, "PATCH", { enabled })
                }
              />
            ),
          },
          {
            title: "操作",
            render: (_, user) => (
              <Space wrap>
                {can("user.update") && (
                  <Button onClick={() => edit(user)}>编辑</Button>
                )}
                {can("user.reset_password") && (
                  <Button
                    onClick={() => setReset({ user, action: "reset-password" })}
                  >
                    重置密码
                  </Button>
                )}
                {can("user.reset_totp") && (
                  <Button
                    onClick={() => setReset({ user, action: "reset-totp" })}
                  >
                    重置验证器
                  </Button>
                )}
                {can("user.delete") && (
                  <Popconfirm
                    title="确认删除此用户？"
                    onConfirm={() => act(`users/${user.id}`, "DELETE")}
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
        title={editing ? "编辑用户" : "新增用户"}
        open={editing !== undefined}
        onCancel={() => setEditing(undefined)}
        onOk={() => form.submit()}
        confirmLoading={busy}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={async (values) => {
            const path = editing ? `users/${editing.id}` : "users";
            if (await act(path, editing ? "PUT" : "POST", values))
              setEditing(undefined);
          }}
        >
          <Form.Item
            name="username"
            label="账号（创建后不可更改）"
            rules={[{ required: true, pattern: /^[a-zA-Z0-9_.-]{1,100}$/ }]}
          >
            <Input disabled={!!editing} autoComplete="off" />
          </Form.Item>
          <Form.Item
            name="displayName"
            label="显示名称"
            rules={[{ required: true, max: 100 }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="roleId" label="角色" rules={[{ required: true }]}>
            <Select
              options={roles
                .filter((role) => role.enabled)
                .map((role) => ({ value: role.id, label: role.name }))}
            />
          </Form.Item>
          {!editing && (
            <Form.Item
              name="password"
              label="初始密码"
              rules={[{ required: true, min: 12, max: 72 }]}
            >
              <Input.Password autoComplete="new-password" />
            </Form.Item>
          )}
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch disabled={!!editing && !can("user.disable")} />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title={reset?.action === "reset-password" ? "重置密码" : "重置验证器"}
        open={!!reset}
        onCancel={() => setReset(undefined)}
        footer={null}
        destroyOnHidden
      >
        <p>
          先在个人安全中心完成近期身份验证。操作会撤销目标用户的全部会话与票据。
        </p>
        <Form
          layout="vertical"
          onFinish={async (values) => {
            if (
              reset &&
              (await act(
                `users/${reset.user.id}/${reset.action}`,
                "POST",
                values,
              ))
            )
              setReset(undefined);
          }}
        >
          {reset?.action === "reset-password" && (
            <Form.Item
              name="password"
              label="新初始密码"
              rules={[{ required: true, min: 12, max: 72 }]}
            >
              <Input.Password autoComplete="new-password" />
            </Form.Item>
          )}
          <Form.Item
            name="reason"
            label="操作原因"
            rules={[{ required: true, max: 500 }]}
          >
            <Input.TextArea />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={busy}>
            确认重置
          </Button>
        </Form>
      </Modal>
    </Card>
  );
}
