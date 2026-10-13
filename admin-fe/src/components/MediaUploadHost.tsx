import { useEffect } from "react";
import {
  Button,
  Drawer,
  List,
  Progress,
  Space,
  Typography,
  FloatButton,
} from "antd";
import { CloudUploadOutlined } from "@ant-design/icons";
import {
  mediaUploads,
  useMediaUploads,
  type UploadState,
} from "./mediaUploadStore";
const labels: Record<UploadState, string> = {
  queued: "等待上传",
  initializing: "准备上传",
  uploading: "上传中",
  paused: "已暂停",
  confirming: "校验图片",
  done: "完成",
  error: "失败",
  canceled: "已取消",
};
export default function MediaUploadHost({ userId }: { userId?: string }) {
  const { tasks, visible } = useMediaUploads();
  useEffect(() => () => mediaUploads.reset(), [userId]);
  useEffect(() => {
    const before = (e: BeforeUnloadEvent) => {
      if (tasks.some((t) => !["done", "canceled"].includes(t.state))) {
        e.preventDefault();
      }
    };
    window.addEventListener("beforeunload", before);
    return () => window.removeEventListener("beforeunload", before);
  }, [tasks]);
  if (!userId || !tasks.length) return null;
  const active = tasks.filter(
    (t) => !["done", "error", "canceled"].includes(t.state),
  ).length;
  return (
    <>
      <FloatButton
        icon={<CloudUploadOutlined />}
        badge={{ count: active }}
        tooltip="图片上传任务"
        onClick={() => mediaUploads.show(true)}
      />
      <Drawer
        title="图片上传"
        open={visible}
        onClose={() => mediaUploads.show(false)}
        size={480}
        extra={<Button onClick={() => mediaUploads.clear()}>清除已结束</Button>}
      >
        <Typography.Paragraph type="secondary">
          关闭面板或切换菜单可继续上传；暂停后继续将重传当前图片。刷新或关闭浏览器会中断队列。
        </Typography.Paragraph>
        <List
          dataSource={tasks}
          renderItem={(task) => (
            <List.Item>
              <div style={{ width: "100%" }}>
                <Typography.Text ellipsis style={{ maxWidth: 330 }}>
                  {task.file.name}
                </Typography.Text>
                <Progress
                  percent={Math.round(task.percent)}
                  status={
                    task.state === "error"
                      ? "exception"
                      : task.state === "done"
                        ? "success"
                        : "normal"
                  }
                />
                <Space wrap>
                  <span>{labels[task.state]}</span>
                  {task.state === "uploading" && (
                    <span>{(task.speed / 1024).toFixed(0)} KiB/s</span>
                  )}
                  {["queued", "uploading"].includes(task.state) && (
                    <Button
                      size="small"
                      onClick={() => mediaUploads.pause(task)}
                    >
                      暂停
                    </Button>
                  )}
                  {["paused", "error"].includes(task.state) && (
                    <Button
                      size="small"
                      onClick={() => mediaUploads.retry(task)}
                    >
                      {task.state === "paused" ? "继续" : "重试"}
                    </Button>
                  )}
                  {!["done", "canceled", "confirming"].includes(task.state) && (
                    <Button
                      size="small"
                      danger
                      onClick={() => void mediaUploads.cancel(task)}
                    >
                      取消
                    </Button>
                  )}
                </Space>
                {task.error && (
                  <Typography.Paragraph type="danger">
                    {task.error}
                  </Typography.Paragraph>
                )}
              </div>
            </List.Item>
          )}
        />
      </Drawer>
    </>
  );
}
