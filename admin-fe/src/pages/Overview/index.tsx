import { useEffect, useState } from "react";
import { Alert, Card, Space, Tag, Typography } from "antd";
import { getReadiness } from "../../services/system";

export default function Overview() {
  const [status, setStatus] = useState("检查中");
  useEffect(() => {
    let active = true;
    getReadiness()
      .then(() => {
        if (active) setStatus("已连接");
      })
      .catch(() => {
        if (active) setStatus("未连接");
      });
    return () => {
      active = false;
    };
  }, []);
  return (
    <Space direction="vertical" size="large" style={{ width: "100%" }}>
      <Alert
        type="info"
        showIcon
        message="系统管理、素材库、文章管理和官网配置已接入后端"
        description="先上传品牌素材，再配置官网 Banner、Logo 和色系；商品管理支持录入和上下架；文章支持富文本，主题与 Banner 支持在线配置。"
      />
      <Card title="工作台">
        <Typography.Paragraph>
          独立站点 · 英文展示站 · 多用户与角色权限 · PostgreSQL
        </Typography.Paragraph>
        <Tag color={status === "已连接" ? "green" : "orange"}>
          后端数据库：{status}
        </Tag>
      </Card>
    </Space>
  );
}
