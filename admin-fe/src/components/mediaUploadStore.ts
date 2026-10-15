import COS from "cos-js-sdk-v5";
import { useSyncExternalStore } from "react";
import { api, APIException, type Media } from "../services/cms";
import {
  getPolicy,
  uploadMIME,
  finishUpload,
  type Credential,
  type UploadBatch,
} from "../services/media";
export type UploadState =
  | "queued"
  | "initializing"
  | "uploading"
  | "paused"
  | "confirming"
  | "done"
  | "error"
  | "canceled";
export interface UploadTask {
  key: string;
  file: File;
  groupId: string | null;
  state: UploadState;
  percent: number;
  speed: number;
  error?: string;
  batch?: UploadBatch;
  uploaded?: boolean;
  media?: Media;
  run: number;
  cancelTransfer?: () => void;
}
let snapshot: { tasks: UploadTask[]; visible: boolean; revision: number } = {
  tasks: [],
  visible: false,
  revision: 0,
};
const listeners = new Set<() => void>();
const emit = () => {
  snapshot = { ...snapshot, tasks: [...snapshot.tasks] };
  listeners.forEach((fn) => fn());
};
const patch = (task: UploadTask, values: Partial<UploadTask>) => {
  Object.assign(task, values);
  emit();
};
export function useMediaUploads() {
  return useSyncExternalStore(
    (fn) => {
      listeners.add(fn);
      return () => {
        listeners.delete(fn);
      };
    },
    () => snapshot,
  );
}
const active = (s: UploadState) =>
  ["initializing", "uploading", "confirming"].includes(s);
function pump() {
  let slots = 3 - snapshot.tasks.filter((t) => active(t.state)).length;
  for (const task of snapshot.tasks)
    if (slots > 0 && task.state === "queued") {
      slots--;
      void run(task);
    }
}
async function run(task: UploadTask) {
  const attempt = ++task.run;
  const live = () => task.run === attempt;
  patch(task, { state: "initializing", error: undefined });
  try {
    if (!task.batch) {
      const policy = await getPolicy();
      if (!policy.enabled) throw new Error("COS 尚未配置");
      const mime = uploadMIME(task.file, policy);
      if (
        !mime ||
        task.file.size > policy.maxImageBytes ||
        task.file.size <= 0
      )
        throw new Error("图片类型或大小不符合当前上传要求");
      if (!live()) return;
      const batch = await api<UploadBatch>("media/upload/init", "POST", {
        groupId: task.groupId,
        files: [
          { name: task.file.name, mime, size: task.file.size },
        ],
      });
      task.batch = batch;
      if (!live()) {
        if (task.state === "canceled")
          await api(`media/${batch.items[0].id}/cancel`, "POST");
        return;
      }
    }
    const batch = task.batch;
    if (!task.uploaded) {
      if (batch.credential.expiredTime * 1000 < Date.now() + 60000)
        batch.credential = await api<Credential>(
          "media/upload/refresh",
          "POST",
          { batchId: batch.batchId },
        );
      if (!live()) return;
      patch(task, { state: "uploading", percent: 0, speed: 0 });
      await new Promise<void>((resolve, reject) => {
        let taskId = "";
        let settled = false;
        const settle = (error?: Error) => {
          if (settled) return;
          settled = true;
          error ? reject(error) : resolve();
        };
        let refreshing: Promise<Credential> | undefined;
        const cos = new COS({
          getAuthorization: (_options, callback) => {
            const authorize = async () => {
              if (!live()) throw new Error("上传已停止");
              if (batch.credential.expiredTime * 1000 < Date.now() + 60000) {
                refreshing ??= api<Credential>("media/upload/refresh", "POST", {
                  batchId: batch.batchId,
                }).finally(() => {
                  refreshing = undefined;
                });
                batch.credential = await refreshing;
              }
              if (!live()) throw new Error("上传已停止");
              const c = batch.credential;
              callback({
                TmpSecretId: c.secretId,
                TmpSecretKey: c.secretKey,
                SecurityToken: c.token,
                StartTime: c.startTime,
                ExpiredTime: c.expiredTime,
              });
            };
            void authorize().catch((error) => {
              settle(error);
              if (taskId) cos.cancelTask(taskId);
            });
          },
        });
        task.cancelTransfer = () => {
          if (taskId) cos.cancelTask(taskId);
          settle(new Error("上传已停止"));
        };
        cos.uploadFile(
          {
            Bucket: batch.bucket,
            Region: batch.region,
            Key: batch.items[0].key,
            Body: task.file,
            ContentType: batch.items[0].mime,
            SliceSize: 65 * 1024 * 1024,
            onTaskReady: (id: string) => {
              taskId = id;
              if (!live()) task.cancelTransfer?.();
            },
            onProgress: (data) => {
              if (live())
                patch(task, {
                  percent: Math.min(100, data.percent * 100),
                  speed: data.speed,
                });
            },
          } as COS.UploadFileParams & { onTaskReady: (id: string) => void },
          (error) =>
            settle(
              error ? new Error("COS 上传失败，请检查网络后重试") : undefined,
            ),
        );
      });
      if (!live()) return;
      task.uploaded = true;
    }
    patch(task, {
      state: "confirming",
      percent: 100,
      cancelTransfer: undefined,
    });
    const media = await finishUpload(batch.items[0].id);
    if (!live()) return;
    patch(task, { state: "done", media });
    snapshot = { ...snapshot, revision: snapshot.revision + 1 };
    emit();
  } catch (error) {
    if (!live()) return;
    if (
      error instanceof APIException &&
      [
        "MEDIA_FILE_INVALID",
        "MEDIA_UPLOAD_EXPIRED",
        "MEDIA_UPLOAD_STATE",
        "NOT_FOUND",
      ].includes(error.code)
    ) {
      task.batch = undefined;
      task.uploaded = false;
    }
    patch(task, {
      state: "error",
      error: (error as Error).message,
      cancelTransfer: undefined,
    });
  } finally {
    if (live()) task.cancelTransfer = undefined;
    pump();
  }
}
export const mediaUploads = {
  add(files: File[], groupId: string | null) {
    if (
      snapshot.tasks.filter((t) => !["done", "canceled"].includes(t.state))
        .length +
        files.length >
      100
    )
      throw new Error("上传队列最多保留 100 个未完成任务");
    snapshot.tasks.push(
      ...files.map((file) => ({
        key: crypto.randomUUID(),
        file,
        groupId,
        state: "queued" as const,
        percent: 0,
        speed: 0,
        run: 0,
      })),
    );
    snapshot.visible = true;
    emit();
    pump();
  },
  show(visible: boolean) {
    snapshot.visible = visible;
    emit();
  },
  pause(task: UploadTask) {
    if (!["uploading", "queued"].includes(task.state)) return;
    task.run++;
    task.cancelTransfer?.();
    patch(task, { state: "paused", cancelTransfer: undefined });
    pump();
  },
  retry(task: UploadTask) {
    if (!["error", "paused"].includes(task.state)) return;
    patch(task, { state: "queued", error: undefined });
    pump();
  },
  async cancel(task: UploadTask) {
    if (["confirming", "done", "canceled"].includes(task.state)) return;
    task.run++;
    task.cancelTransfer?.();
    patch(task, { state: "canceled", cancelTransfer: undefined });
    pump();
    if (task.batch) {
      try {
        await api(`media/${task.batch.items[0].id}/cancel`, "POST");
      } catch (e) {
        patch(task, { state: "error", error: (e as Error).message });
      }
    }
  },
  clear() {
    snapshot.tasks = snapshot.tasks.filter(
      (t) => !["done", "canceled"].includes(t.state),
    );
    emit();
  },
  reset() {
    for (const task of snapshot.tasks) {
      task.run++;
      task.cancelTransfer?.();
    }
    snapshot = { tasks: [], visible: false, revision: snapshot.revision + 1 };
    emit();
  },
};
