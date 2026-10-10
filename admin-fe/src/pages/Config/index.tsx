import { useEffect, useState } from "react";
import { Alert, Button, Card, Descriptions, Table } from "antd";
import { configStatus, type ConfigStatus } from "../../services/config";
export default function ConfigPage() {
  const [status, setStatus] = useState<ConfigStatus>();
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  async function load() {
    setBusy(true);
    try {
      setStatus(await configStatus());
      setError("");
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  useEffect(() => {
    void load();
    const timer = setInterval(() => void load(), 10000);
    return () => clearInterval(timer);
  }, []);
  return (
    <Card
      title="运行配置"
      extra={
        <Button loading={busy} onClick={load}>
          刷新
        </Button>
      }
    >
      {error && <Alert type="error" title={error} />}
      {status && (
        <>
          {status.error && (
            <Alert
              type="warning"
              title="最近配置更新失败，继续使用上次有效版本"
              description={status.error}
            />
          )}
          <Descriptions
            items={[
              { key: "version", label: "有效版本", children: status.version },
              {
                key: "loaded",
                label: "最近成功",
                children: new Date(status.loadedAt).toLocaleString("zh-CN", {
                  timeZone: "Asia/Shanghai",
                }),
              },
              {
                key: "checked",
                label: "最近检查",
                children: new Date(status.checkedAt).toLocaleString("zh-CN", {
                  timeZone: "Asia/Shanghai",
                }),
              },
            ]}
          />
          <Table
            pagination={false}
            rowKey="path"
            dataSource={Object.entries(status.sources).map(
              ([path, source]) => ({ path, source }),
            )}
            columns={[
              { title: "字段", dataIndex: "path" },
              {
                title: "有效来源",
                dataIndex: "source",
                render: (value: string) =>
                  value.startsWith("environment:")
                    ? `环境变量 ${value.slice(12)}（覆盖 YAML）`
                    : value === "yaml"
                      ? "YAML"
                      : "默认值",
              },
            ]}
          />
        </>
      )}
    </Card>
  );
}
