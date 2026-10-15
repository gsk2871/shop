import { api, type Media } from "./cms";
export interface MediaGroup {
  id: string;
  name: string;
  count: number;
}
export interface UploadPolicy {
  version: number;
  enabled: boolean;
  allowedTypes: string[];
  allowedExtensions?: Record<string, string>;
  maxImageBytes: number;
  maxBatchFiles: number;
}
export function uploadMIME(file: File, policy: UploadPolicy): string | undefined {
  if (policy.allowedTypes.includes(file.type)) return file.type;
  const ext = file.name.match(/\.[^.]+$/)?.[0].toLowerCase();
  const candidate = ext ? policy.allowedExtensions?.[ext] : undefined;
  return candidate && policy.allowedTypes.includes(candidate)
    ? candidate
    : undefined;
}
export interface Credential {
  secretId: string;
  secretKey: string;
  token: string;
  startTime: number;
  expiredTime: number;
}
export interface UploadBatch {
  batchId: string;
  bucket: string;
  region: string;
  credential: Credential;
  items: { id: string; key: string; mime: string; name: string }[];
}
export const getGroups = () => api<{ items: MediaGroup[] }>("media/groups");
export const getPolicy = () => api<UploadPolicy>("media/upload-policy");
export const finishUpload = (id: string) =>
  api<Media>(`media/${id}/complete`, "POST");
