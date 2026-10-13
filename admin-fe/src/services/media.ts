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
  maxImageBytes: number;
  maxBatchFiles: number;
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
