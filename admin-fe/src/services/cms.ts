export type { Banner, Settings } from "./contracts";
export type { Article } from "./contracts";
export interface Media {
  id: string;
  type: "image" | "video" | "document";
  available: boolean;
  name: string;
  url: string;
  mime: string;
  size: number;
  createdAt: string;
}
export async function api<T>(
  path: string,
  method = "GET",
  body?: unknown,
): Promise<T> {
  const response = await fetch(`/backend/admin/api/v1/${path}`, {
    method,
    credentials: "same-origin",
    headers: {
      "X-Requested-With": "peptide-admin",
      ...(body ? { "Content-Type": "application/json" } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
    signal: AbortSignal.timeout(15000),
  });
  const data = await response.json();
  if (!response.ok)
    throw new APIException(
      response.status,
      data.code || "INTERNAL_ERROR",
      data.requestId || "",
    );
  return data;
}
export function assetURL(url: string) {
  return url.startsWith("/uploads/") ? `/backend${url}` : url;
}

export type { Product } from "./contracts";

const errors: Record<string, string> = {
  INVALID_ARGUMENT: "输入格式或业务规则不正确",
  AUTHENTICATION_REQUIRED: "登录已失效，请重新登录",
  INVALID_CREDENTIALS: "账号或验证信息不正确",
  MFA_INVALID: "验证码无效或已使用",
  MFA_TICKET_INVALID: "登录票据已失效，请重新输入密码",
  PERMISSION_DENIED: "没有操作权限",
  ONBOARDING_REQUIRED: "请先完成改密和验证器绑定",
  REAUTHENTICATION_REQUIRED: "请在安全中心再次验证身份，并填写操作原因",
  NOT_FOUND: "内容不存在",
  UPDATE_CONFLICT: "配置已被其他人修改，请刷新后重新确认",
  COUNTRY_ALREADY_ASSIGNED: "此国家已被其他配置占用",
  DUPLICATE_WHATSAPP_ACCOUNT: "同一配置不能包含重复号码",
  DEFAULT_GROUP_PROTECTED:
    "不能停用或删除默认配置及其最后一个可用账号，请先更换默认",
  DEFAULT_GROUP_UNAVAILABLE: "默认配置必须启用且至少有一个带二维码的启用账号",
  VERSION_CONFLICT: "内容已被修改，请刷新后重试",
  CATEGORY_IN_USE: "分类正在被商品使用，请先调整商品分类再删除",
  MEDIA_IN_USE: "素材正在使用，不能删除",
  CONFLICT: "操作冲突，请刷新后重试",
  LAST_SUPER_ADMIN: "不能停用、删除或降级最后一位有效超级管理员",
  RATE_LIMITED: "操作过于频繁，请稍后重试",
  SERVICE_UNAVAILABLE: "服务暂时不可用，请稍后重试",
};
export class APIException extends Error {
  constructor(
    public status: number,
    public code: string,
    public requestId: string,
  ) {
    super(errors[code] || "请求失败，请稍后重试");
  }
}
