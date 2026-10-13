import { useEffect, useState } from "react";
import { Button, Upload, message } from "antd";
import { getPolicy, type UploadPolicy } from "../services/media";
import type { Media } from "../services/cms";
import { mediaUploads, useMediaUploads } from "./mediaUploadStore";
export function useUploadPolicy() {
  const [policy, setPolicy] = useState<UploadPolicy>();
  const [error, setError] = useState("");
  useEffect(() => {
    let live = true;
    const load = () =>
      getPolicy()
        .then((p) => {
          if (live) {
            setPolicy(p);
            setError("");
          }
        })
        .catch((e) => {
          if (live) setError(e.message);
        });
    void load();
    const timer = setInterval(load, 30000);
    return () => {
      live = false;
      clearInterval(timer);
    };
  }, []);
  return { policy, error };
}
export default function MediaUpload({
  groupId = null,
  onUploaded,
}: {
  groupId?: string | null;
  onUploaded?: (media: Media) => void;
}) {
  const { policy, error } = useUploadPolicy();
  const { revision, tasks } = useMediaUploads();
  const [msg, context] = message.useMessage();
  useEffect(() => {
    const last = tasks.filter((t) => t.state === "done").at(-1)?.media;
    if (last) onUploaded?.(last);
  }, [revision]);
  return (
    <>
      {context}
      <Upload
        accept={policy?.allowedTypes.join(",")}
        multiple
        showUploadList={false}
        disabled={!policy?.enabled}
        beforeUpload={(file) => {
          if (!policy?.enabled) return Upload.LIST_IGNORE;
          if (
            !policy.allowedTypes.includes(file.type) ||
            file.size > policy.maxImageBytes ||
            file.size === 0
          ) {
            msg.error(
              `仅支持图片，单张不超过 ${(policy.maxImageBytes / 1024 / 1024).toFixed(0)} MiB`,
            );
            return Upload.LIST_IGNORE;
          }
          try {
            mediaUploads.add([file], groupId);
          } catch (e) {
            msg.error((e as Error).message);
          }
          return Upload.LIST_IGNORE;
        }}
      >
        <Button
          type="primary"
          disabled={!policy?.enabled}
          title={error || (!policy?.enabled ? "请先配置 COS" : "")}
        >
          上传图片
        </Button>
      </Upload>
    </>
  );
}
