import type { ReactNode } from "react";
import { PageContainer } from "@ant-design/pro-components";
import { Alert, Card, Typography } from "antd";

export default function ContentPage({
  title,
  description,
  extra,
  filters,
  error,
  children,
}: {
  title: string;
  description: string;
  extra?: ReactNode;
  filters?: ReactNode;
  error?: string;
  children: ReactNode;
}) {
  return (
    <PageContainer
      title={title}
      subTitle={description}
      breadcrumb={{ items: [] }}
      extra={extra}
    >
      {filters && <Card className="content-filters">{filters}</Card>}
      {error && (
        <Alert
          className="content-error"
          type="error"
          showIcon
          title="列表加载失败"
          description={error}
        />
      )}
      <Card className="content-list">{children}</Card>
    </PageContainer>
  );
}

export function ContentHeading({
  title,
  description,
}: {
  title: string;
  description?: string;
}) {
  return (
    <div className="content-section-heading">
      <Typography.Title level={5}>{title}</Typography.Title>
      {description && (
        <Typography.Text type="secondary">{description}</Typography.Text>
      )}
    </div>
  );
}

export function formatTime(value?: string | null) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(date);
}
