import { useEffect, useState } from "react";
import { Button, Card, Table, message } from "antd";
import { api } from "../../services/cms";
import type { Page } from "../../services/contracts";
interface Audit {
  id: string;
  userId: string | null;
  action: string;
  targetType: string;
  targetId: string;
  result: string;
  requestId: string;
  ip: string;
  createdAt: string;
}
const labels: Record<string, string> = {
  "auth.password": "密码验证",
  "auth.login": "登录成功",
  "auth.totp": "验证码验证",
  "auth.totp.setup": "生成绑定",
  "auth.totp.activate": "绑定验证器",
  "auth.reauthenticate": "近期身份验证",
  "user.create": "创建用户",
  "user.update": "编辑用户",
  "user.status": "启停用户",
  "user.delete": "删除用户",
  "user.password": "修改密码",
  "user.reset-password": "重置密码",
  "user.reset-totp": "重置验证器",
  "user.totp.recover": "服务器恢复验证器",
  "role.create": "创建角色",
  "role.update": "编辑角色",
  "role.delete": "删除角色",
  "role.grant": "角色授权",
};
export default function AuditPage() {
  const [data, setData] = useState<Page<Audit>>({
    items: [],
    total: 0,
    page: 1,
    pageSize: 20,
  });
  const [busy, setBusy] = useState(false);
  const [msg, context] = message.useMessage();
  async function load(page = data.page, pageSize = data.pageSize) {
    setBusy(true);
    try {
      setData(await api(`audit?page=${page}&pageSize=${pageSize}`));
    } catch (e) {
      msg.error((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  useEffect(() => {
    void load();
  }, []);
  return (
    <Card
      title="操作日志"
      extra={<Button onClick={() => void load()}>刷新</Button>}
    >
      {context}
      <Table<Audit>
        rowKey="id"
        loading={busy}
        dataSource={data.items}
        pagination={{
          current: data.page,
          pageSize: data.pageSize,
          total: data.total,
          onChange: load,
        }}
        columns={[
          { title: "操作人", dataIndex: "userId" },
          { title: "操作", render: (_, r) => labels[r.action] || r.action },
          { title: "目标", render: (_, r) => `${r.targetType} ${r.targetId}` },
          {
            title: "结果",
            render: (_, r) => (r.result === "success" ? "成功" : "失败"),
          },
          { title: "请求标识", dataIndex: "requestId" },
          { title: "IP", dataIndex: "ip" },
          {
            title: "时间",
            render: (_, r) =>
              new Date(r.createdAt).toLocaleString("zh-CN", {
                timeZone: "Asia/Shanghai",
              }),
          },
        ]}
      />
    </Card>
  );
}
