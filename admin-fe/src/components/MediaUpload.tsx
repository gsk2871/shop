import { useEffect, useState } from "react";
import { Button, Upload, message } from "antd";
import { api, APIException, type Media } from "../services/cms";
export interface UploadPolicy {
  version: number;
  allowedTypes: string[];
  maxImageBytes: number;
  maxVideoBytes: number;
}
export function useUploadPolicy() {
  const [policy, setPolicy] = useState<UploadPolicy>();
  const [error, setError] = useState("");
  useEffect(() => {
    let active = true;
    const load = () =>
      api<UploadPolicy>("media/upload-policy")
        .then((value) => {
          if (active) {
            setPolicy(value);
            setError("");
          }
        })
        .catch((e: Error) => {
          if (active) setError(e.message);
        });
    void load();
    const timer = setInterval(() => void load(), 10000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, []);
  return { policy, error };
}
export default function MediaUpload({
  onUploaded,
}: {
  onUploaded: (media: Media) => void;
}) {
  const { policy, error } = useUploadPolicy();
  const [busy, setBusy] = useState(false);
  const [msg, context] = message.useMessage();
  return (
    <>
      {context}
      <Upload
        accept={policy?.allowedTypes.join(",")}
        disabled={!policy || busy}
        beforeUpload={(file) => {
          if (!policy) return false;
          if (file.type && !policy.allowedTypes.includes(file.type)) {
            msg.error("此文件类型不在当前上传策略中");
            return Upload.LIST_IGNORE;
          }
          const limit = file.type.startsWith("video/")
            ? policy.maxVideoBytes
            : policy.maxImageBytes;
          if (file.size > limit) {
            msg.error(
              `文件超过当前限制 ${(limit / 1024 / 1024).toFixed(0)} MiB`,
            );
            return Upload.LIST_IGNORE;
          }
          return true;
        }}
        customRequest={({ file, onSuccess, onError, onProgress }) => {
          setBusy(true);
          const xhr = new XMLHttpRequest();
          xhr.open("POST", "/backend/admin/api/v1/media");
          xhr.setRequestHeader("X-Requested-With", "peptide-admin");
          xhr.upload.onprogress = (event) => {
            if (event.lengthComputable)
              onProgress?.({ percent: (event.loaded / event.total) * 100 });
          };
          xhr.onload = () => {
            setBusy(false);
            try {
              const data = JSON.parse(xhr.responseText);
              if (xhr.status < 200 || xhr.status >= 300)
                throw new APIException(
                  xhr.status,
                  data.code || "INTERNAL_ERROR",
                  data.requestId || "",
                );
              onSuccess?.(data);
              onUploaded(data);
              msg.success("上传成功");
            } catch (e) {
              onError?.(e as Error);
              msg.error((e as Error).message);
            }
          };
          xhr.onerror = () => {
            setBusy(false);
            const error = new Error("上传连接中断，请重试");
            onError?.(error);
            msg.error(error.message);
          };
          xhr.onabort = () => setBusy(false);
          const form = new FormData();
          form.append("file", file as File);
          xhr.send(form);
          return { abort: () => xhr.abort() };
        }}
      >
        <Button loading={busy} title={error} disabled={!policy}>
          上传素材
        </Button>
      </Upload>
    </>
  );
}
