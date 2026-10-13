import { useEffect, useState } from "react";
import { useModel } from "@umijs/max";
import {
  Alert,
  Button,
  Card,
  Image,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  message,
} from "antd";
import { api, assetURL, type Media } from "../../services/cms";
import { allows } from "../../services/auth";
import type { Page } from "../../services/contracts";
import MediaUpload, { useUploadPolicy } from "../../components/MediaUpload";
interface Reference {
  id: string;
  ownerType: string;
  ownerId: string;
  field: string;
}
export default function MediaPage() {
  const { initialState } = useModel("@@initialState");
  const can = (p: string) => allows(initialState?.identity, p);
  const { policy, error } = useUploadPolicy();
  const [data, setData] = useState<Page<Media>>({
    items: [],
    total: 0,
    page: 1,
    pageSize: 20,
  });
  const [kind, setKind] = useState("");
  const [query, setQuery] = useState("");
  const [busy, setBusy] = useState(false);
  const [rename, setRename] = useState<Media>();
  const [name, setName] = useState("");
  const [references, setReferences] = useState<Reference[]>();
  const [msg, context] = message.useMessage();
  async function load(
    page = data.page,
    pageSize = data.pageSize,
    type = kind,
    q = query,
  ) {
    setBusy(true);
    try {
      setData(
        await api(
          `media?page=${page}&pageSize=${pageSize}&type=${encodeURIComponent(type)}&q=${encodeURIComponent(q)}`,
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
  }, []);
  return (
    <Card
      title="媒体库"
      extra={
        <Space>
          <Button onClick={() => void load()}>刷新</Button>
          {can("media.upload") && (
            <MediaUpload onUploaded={() => void load(1)} />
          )}
        </Space>
      }
    >
      {context}
      {error && <Alert type="error" title={error} />}
      <p>
        {policy
          ? `上传策略版本 ${policy.version}；图片 ${(policy.maxImageBytes / 1024 / 1024).toFixed(0)} MiB，视频 ${(policy.maxVideoBytes / 1024 / 1024).toFixed(0)} MiB。`
          : "正在读取上传限制…"}
      </p>
      <Space>
        <Select
          style={{ width: 140 }}
          value={kind}
          options={[
            { value: "", label: "全部类型" },
            { value: "image", label: "图片" },
            { value: "video", label: "视频" },
          ]}
          onChange={(value) => {
            setKind(value);
            void load(1, data.pageSize, value);
          }}
        />
        <Input.Search
          placeholder="搜索名称"
          onSearch={(value) => {
            setQuery(value);
            void load(1, data.pageSize, kind, value);
          }}
        />
      </Space>
      <Table<Media>
        loading={busy}
        rowKey="id"
        dataSource={data.items}
        pagination={{
          current: data.page,
          pageSize: data.pageSize,
          total: data.total,
          onChange: load,
          showSizeChanger: true,
        }}
        columns={[
          {
            title: "预览",
            render: (_, row) =>
              row.type === "image" ? (
                <Image width={90} src={assetURL(row.url)} />
              ) : row.type === "video" ? (
                <video
                  controls
                  preload="metadata"
                  src={assetURL(row.url)}
                  style={{ width: 180, maxWidth: "100%" }}
                />
              ) : null,
          },
          { title: "名称", dataIndex: "name" },
          { title: "类型", dataIndex: "mime" },
          {
            title: "大小",
            render: (_, row) => `${(row.size / 1024 / 1024).toFixed(2)} MiB`,
          },
          {
            title: "上传时间",
            render: (_, row) =>
              new Date(row.createdAt).toLocaleString("zh-CN", {
                timeZone: "Asia/Shanghai",
              }),
          },
          {
            title: "操作",
            render: (_, row) => (
              <Space>
                {can("media.update") && (
                  <Button
                    onClick={() => {
                      setRename(row);
                      setName(row.name);
                    }}
                  >
                    重命名
                  </Button>
                )}
                <Button
                  onClick={async () => {
                    try {
                      setReferences(
                        (
                          await api<{ items: Reference[] }>(
                            `media/${row.id}/references`,
                          )
                        ).items,
                      );
                    } catch (e) {
                      msg.error((e as Error).message);
                    }
                  }}
                >
                  引用位置
                </Button>
                {can("media.delete") && (
                  <Popconfirm
                    title="删除素材？被引用素材无法删除。"
                    onConfirm={async () => {
                      try {
                        await api(`media/${row.id}`, "DELETE");
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
        title="重命名素材"
        open={!!rename}
        onCancel={() => setRename(undefined)}
        onOk={async () => {
          if (!rename) return;
          try {
            await api(`media/${rename.id}`, "PATCH", { name });
            setRename(undefined);
            await load();
          } catch (e) {
            msg.error((e as Error).message);
          }
        }}
      >
        <Input
          maxLength={255}
          value={name}
          onChange={(event) => setName(event.target.value)}
        />
      </Modal>
      <Modal
        title="引用位置"
        open={references !== undefined}
        onCancel={() => setReferences(undefined)}
        footer={null}
      >
        <Table
          rowKey="id"
          dataSource={references}
          columns={[
            { title: "对象", dataIndex: "ownerType" },
            { title: "标识", dataIndex: "ownerId" },
            { title: "字段", dataIndex: "field" },
          ]}
        />
      </Modal>
    </Card>
  );
}
