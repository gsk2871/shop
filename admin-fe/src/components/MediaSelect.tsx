import { useEffect, useState } from "react";
import { Button, Image, Modal, Space, Typography } from "antd";
import { api, assetURL, type Media } from "../services/cms";
import MediaLibrary from "./MediaLibrary";
export default function MediaSelect({
  value,
  onChange,
  onSelect,
}: {
  value?: string;
  onChange?: (id: string) => void;
  onSelect?: (media: Media) => void;
}) {
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState<Media>();
  const [error, setError] = useState("");
  useEffect(() => {
    let live = true;
    setSelected(undefined);
    setError("");
    if (value)
      api<Media>(`media/${value}`)
        .then((m) => {
          if (live) setSelected(m);
        })
        .catch((e) => {
          if (live) setError(e.message);
        });
    return () => {
      live = false;
    };
  }, [value]);
  return (
    <>
      <Space>
        <Button onClick={() => setOpen(true)}>
          {value ? "更换图片" : "从媒体库选择"}
        </Button>
        {value && <Button onClick={() => onChange?.("")}>清除</Button>}
      </Space>
      {selected && (
        <div style={{ marginTop: 10 }}>
          <Image width={120} src={assetURL(selected.url)} alt={selected.name} />
          <div>{selected.name}</div>
        </div>
      )}
      {error && (
        <Typography.Paragraph type="danger">
          {error}，请选择其他图片。
        </Typography.Paragraph>
      )}
      <Modal
        title="选择图片"
        open={open}
        onCancel={() => setOpen(false)}
        footer={null}
        width={1100}
        destroyOnHidden
      >
        {open && (
          <MediaLibrary
            onSelect={(media) => {
              onChange?.(media.id);
              onSelect?.(media);
              setOpen(false);
            }}
          />
        )}
      </Modal>
    </>
  );
}
