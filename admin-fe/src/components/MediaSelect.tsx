import { useEffect, useState } from "react";
import { useModel } from "@umijs/max";
import {
  Button,
  Image,
  Input,
  Modal,
  Select,
  Space,
  Table,
  message,
} from "antd";
import { api, assetURL, type Media } from "../services/cms";
import { allows } from "../services/auth";
import type { Page } from "../services/contracts";
import MediaUpload from "./MediaUpload";
export default function MediaSelect({
  value,
  onChange,
  onSelect,
  types = ["image"],
}: {
  value?: string;
  onChange?: (id: string) => void;
  onSelect?: (media: Media) => void;
  types?: ("image" | "video")[];
}) {
  const { initialState } = useModel("@@initialState");
  const [open, setOpen] = useState(false);
  const [data, setData] = useState<Page<Media>>({
    items: [],
    total: 0,
    page: 1,
    pageSize: 20,
  });
  const [kind, setKind] = useState<string>(types.length === 1 ? types[0] : "");
  const [query, setQuery] = useState("");
  const [msg, context] = message.useMessage();
  async function load(page = 1, type = kind, q = query) {
    try {
      setData(
        await api(
          `media?page=${page}&pageSize=20&type=${encodeURIComponent(type)}&q=${encodeURIComponent(q)}`,
        ),
      );
    } catch (e) {
      msg.error((e as Error).message);
    }
  }
  useEffect(() => {
    if (open) void load();
  }, [open]);
  return (
    <>
      {context}
      <Space>
        <Button onClick={() => setOpen(true)}>
          {value ? `已选择素材 #${value}` : "从媒体库选择"}
        </Button>
        {value && <Button onClick={() => onChange?.("")}>清除</Button>}
      </Space>
      <Modal
        title="选择媒体素材"
        open={open}
        onCancel={() => setOpen(false)}
        footer={null}
        width={760}
        destroyOnHidden
      >
        <Space>
          {types.length > 1 && (
            <Select
              value={kind}
              options={[
                { value: "", label: "图片与视频" },
                ...types.map((type) => ({
                  value: type,
                  label: type === "image" ? "图片" : "视频",
                })),
              ]}
              onChange={(value) => {
                setKind(value);
                void load(1, value);
              }}
            />
          )}
          <Input.Search
            placeholder="搜索名称"
            onSearch={(value) => {
              setQuery(value);
              void load(1, kind, value);
            }}
          />
          {allows(initialState?.identity, "media.upload") && (
            <MediaUpload onUploaded={() => void load()} />
          )}
        </Space>
        <Table<Media>
          rowKey="id"
          dataSource={data.items}
          pagination={{
            current: data.page,
            pageSize: data.pageSize,
            total: data.total,
            onChange: (page) => void load(page),
          }}
          columns={[
            {
              title: "预览",
              render: (_, row) =>
                row.type === "image" ? (
                  <Image width={72} src={assetURL(row.url)} />
                ) : row.type === "video" ? (
                  <video
                    controls
                    preload="metadata"
                    src={assetURL(row.url)}
                    width={140}
                  />
                ) : null,
            },
            { title: "名称", dataIndex: "name" },
            {
              title: "选择",
              render: (_, row) => (
                <Button
                  disabled={!types.includes(row.type as "image" | "video")}
                  onClick={() => {
                    onChange?.(row.id);
                    onSelect?.(row);
                    setOpen(false);
                  }}
                >
                  选择
                </Button>
              ),
            },
          ]}
        />
      </Modal>
    </>
  );
}
