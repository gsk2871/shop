import { Button, Space, Typography } from "antd";
import MediaSelect from "./MediaSelect";

export default function MediaMultiSelect({
  value = [],
  onChange,
}: {
  value?: string[];
  onChange?: (ids: string[]) => void;
}) {
  function move(index: number, offset: number) {
    const next = [...value];
    [next[index], next[index + offset]] = [next[index + offset], next[index]];
    onChange?.(next);
  }
  return (
    <Space orientation="vertical" style={{ width: "100%" }}>
      <Typography.Text type="secondary">
        按顺序展示，最多 20 张；产品第一张图片用作封面。
      </Typography.Text>
      {value.map((id, index) => (
        <div key={id} style={{ border: "1px solid #eee", padding: 12 }}>
          <MediaSelect
            value={id}
            onChange={(next) => {
              if (!next) onChange?.(value.filter((_, i) => i !== index));
              else if (!value.includes(next))
                onChange?.(value.map((old, i) => (i === index ? next : old)));
            }}
          />
          <Space style={{ marginTop: 8 }}>
            <Button disabled={index === 0} onClick={() => move(index, -1)}>
              前移
            </Button>
            <Button
              disabled={index === value.length - 1}
              onClick={() => move(index, 1)}
            >
              后移
            </Button>
          </Space>
        </div>
      ))}
      {value.length < 20 && (
        <MediaSelect
          onChange={(id) => {
            if (id && !value.includes(id)) onChange?.([...value, id]);
          }}
        />
      )}
    </Space>
  );
}
