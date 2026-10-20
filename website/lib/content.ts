import "server-only";
import { defaultTheme, type Settings, type PublicSocialAccount } from "./contracts";
export type {
  PublicArticle as Article,
  Settings,
  PublicBanner as Banner,
} from "./contracts";
type SiteSettings = Settings & { socialAccounts?: PublicSocialAccount[] };
export const defaults: SiteSettings = {
  socialAccounts: [],
  googleAds: { enabled: false, id: "", conversionLabel: "" },
  brand: "",
  brandMark: "",
  homeTitle: "",
  homeContent: "",
  aboutTitle: "",
  aboutContent: "",
  privacyTitle: "",
  privacyContent: "",
  productsTitle: "",
  researchTitle: "",
  newsTitle: "",
  cookieTitle: "",
  cookieContent: "",

  logo: "",
  logoMediaId: "",
  theme: defaultTheme,
  description: "",
  email: "",
  whatsapp: "",
  disclaimer: "",
  version: 1,
};
export interface ContentError {
  status: number;
  code: string;
  requestId: string;
}
export async function contentResult<T>(
  path: string,
): Promise<{ data: T | null; error: ContentError | null }> {
  try {
    const response = await fetch(
      `${process.env.BACKEND_URL || "http://127.0.0.1:8080"}/api/v1/${path}`,
      {
        cache: "no-store",
        credentials: "omit",
        signal: AbortSignal.timeout(3000),
      },
    );
    if (!response.ok) {
      let body: { code?: string; requestId?: string } = {};
      try {
        body = await response.json();
      } catch {}
      return {
        data: null,
        error: {
          status: response.status,
          code: body.code || "SERVICE_UNAVAILABLE",
          requestId: body.requestId || "",
        },
      };
    }
    return { data: await response.json(), error: null };
  } catch {
    return {
      data: null,
      error: { status: 503, code: "SERVICE_UNAVAILABLE", requestId: "" },
    };
  }
}
export async function content<T>(path: string): Promise<T | null> {
  return (await contentResult<T>(path)).data;
}
export async function settings() {
  const s = await content<SiteSettings>("site");
  return { ...defaults, ...s, theme: { ...defaultTheme, ...s?.theme } };
}

export async function banners(slot: string) {
  return (
    (
      await content<{ items: import("./contracts").PublicBanner[] }>(
        `banners?slot=${encodeURIComponent(slot)}`,
      )
    )?.items || []
  );
}
