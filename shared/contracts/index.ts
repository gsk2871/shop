/** IDs use decimal strings because PostgreSQL BIGINT can exceed JS safe integers. */
export type EntityID = string;

export interface APIError {
  code: string;
  message: string;
  requestId: string;
}

export interface Page<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

/** The backend converts and formats prices. Clients do not multiply or divide. */
export interface Price {
  currency: string;
  salePrice: string;
  displayPrice: string;
}

/** Media IDs must match the controlled nodes in HTML; the server verifies both. */
export interface RichText {
  html: string;
  mediaIds: EntityID[];
}

export interface VersionedWrite {
  expectedVersion: number;
}

/** Administrative metadata stays inside the management application. */
export interface PermissionNode {
  code: string;
  name: string;
  type: "directory" | "menu" | "action";
  route: string | null;
  permissionCode: string | null;
  sort: number;
  children: PermissionNode[];
}

export interface SpecValue {
  id?: EntityID;
  key: string;
  name: string;
  sort: number;
}
export interface ProductSpec {
  id?: EntityID;
  key: string;
  name: string;
  sort: number;
  values: SpecValue[];
}
export interface ProductSKU {
  id?: EntityID;
  key: string;
  enabled: boolean;
  sort: number;
  specValueKeys: string[];
  currency: string;
  salePrice: string | null;
  price?: Price | null;
}
export interface PublicSpecValue {
  id?: EntityID;
  key: string;
  name: string;
}
export interface PublicProductSpec {
  id?: EntityID;
  key: string;
  name: string;
  values: PublicSpecValue[];
}
export interface PublicSKU {
  id?: EntityID;
  key: string;
  specValueKeys: string[];
  price: Price;
}
export interface PublicProduct {
  images?: string[];
  reports?: string[];
  image: string;
  report: string;
  id?: EntityID;
  name: string;
  slug: string;
  categoryId: EntityID | "";
  category: string;
  summary: string;
  detail: RichText;
  specs: PublicProductSpec[];
  skus: PublicSKU[];
  fromPrice?: Price | null;
}
export interface Product extends Omit<PublicProduct, "specs" | "skus"> {
  homeFeatured: boolean;
  homeSort: number;
  imageMediaIds?: EntityID[];
  reportMediaIds?: EntityID[];
  imageMediaId: EntityID | "";
  reportMediaId: EntityID | "";
  specs: ProductSpec[];
  skus: ProductSKU[];
  id: EntityID;
  status: "draft" | "published" | "offline";
  version: number;
  expectedVersion?: number;
  updatedAt: string;
}
export interface ProductSummary {
  homeFeatured?: boolean;
  homeSort?: number;
  image: string;
  id: EntityID;
  name: string;
  slug: string;
  categoryId: EntityID | "";
  category: string;
  summary: string;
  fromPrice?: Price | null;
  status?: "draft" | "published" | "offline";
  version?: number;
  updatedAt?: string;
}

export interface PublicArticle {
  id?: EntityID;
  title: string;
  slug: string;
  summary: string;
  body: RichText;
  cover: string;
  firstPublishedAt: string | null;
}
export interface Article extends PublicArticle {
  id: EntityID;
  coverMediaId: string;
  status: "draft" | "published" | "offline";
  pinned: boolean;
  sort: number;
  version: number;
  expectedVersion?: number;
  updatedAt: string;
}

export interface Theme {
  primary: string;
  background: string;
  surface: string;
  text: string;
  muted: string;
  border: string;
  accent: string;
  onPrimary: string;
}
export interface PublicSocialAccount {
  name: string;
  platform: "facebook" | "tiktok" | "youtube" | "x" | "instagram" | "linkedin" | "other";
  link: string;
  icon: string;
}
export interface SocialAccount extends PublicSocialAccount {
  id: EntityID;
  version: number;
  expectedVersion?: number;
  sort: number;
  iconMediaId: string;
  enabled: boolean;
}
export interface GoogleAdsSettings {
  enabled: boolean;
  id: string;
  conversionLabel: string;
}
export interface Settings {
  googleAds: GoogleAdsSettings;
  brandMark: string;
  homeTitle: string;
  homeContent: string;
  aboutTitle: string;
  aboutContent: string;
  privacyTitle: string;
  privacyContent: string;
  productsTitle: string;
  researchTitle: string;
  newsTitle: string;
  cookieTitle: string;
  cookieContent: string;

  brand: string;
  logoMediaId: string;
  logo: string;
  theme: Theme;
  description: string;
  email: string;
  whatsapp: string;
  disclaimer: string;
  version: number;
  expectedVersion?: number;
}
export interface PublicBanner {
  id: string;
  title: string;
  subtitle: string;
  image: string;
  mobileImage: string;
  ctaText: string;
  link: string;
}
export interface Banner extends PublicBanner {
  productId: string;
  productName: string;
  slotCode: string;
  imageMediaId: string;
  mobileImageMediaId: string;
  sort: number;
  enabled: boolean;
  startsAt: string | null;
  endsAt: string | null;
  version: number;
  expectedVersion?: number;
}
export const defaultTheme: Theme = {
  primary: "#174e42",
  background: "#f6f5ef",
  surface: "#ffffff",
  text: "#1c2825",
  muted: "#65706b",
  border: "#d9ddd4",
  accent: "#d39d49",
  onPrimary: "#ffffff",
};

export interface WhatsAppAccount {
  greeting: string;
  id: EntityID;
  name: string;
  phoneNumber: string;
  qrCodeMediaId: EntityID;
  qrCodeUrl: string;
  enabled: boolean;
  allocationCount: number;
  updatedAt: string;
}
export interface WhatsAppAccountInput {
  greeting?: string;
  id?: EntityID;
  name: string;
  phoneNumber: string;
  qrCodeMediaId: EntityID;
  enabled: boolean;
}
export interface WhatsAppGroup {
  id: EntityID;
  name: string;
  countryCodes: string[];
  enabled: boolean;
  updatedAt: string;
  isDefault: boolean;
  pendingSetup: boolean;
  accountCount: number;
  availableAccountCount: number;
  accounts?: WhatsAppAccount[];
}
export interface WhatsAppGroupInput {
  name: string;
  countryCodes: string[];
  enabled: boolean;
  expectedUpdatedAt?: string;
  accounts: WhatsAppAccountInput[];
}
export interface WhatsAppGroupPage extends Page<WhatsAppGroup> {
  defaultGroupId: EntityID | "";
  defaultGroupName: string;
  routingUpdatedAt: string;
  geoipAvailable: boolean;
}
export interface WhatsAppContact {
  accountName: string;
  phoneNumber: string;
  qrCodeUrl: string;
  url: string;
}

export interface PublicCategory {
  cover?: string;
  id: EntityID;
  name: string;
}
export interface Category extends PublicCategory {
  coverMediaId?: EntityID | "";
  sort: number;
  version: number;
  expectedVersion?: number;
}

export interface PublicFAQ {
  id: EntityID;
  question: string;
  answer: string;
}
export interface FAQ extends PublicFAQ {
  sort: number;
  enabled: boolean;
  version: number;
  expectedVersion?: number;
}

export interface LeadPopupConfig {
 enabled: boolean;
 title: string;
 description: string;
 placeholder: string;
 submitText: string;
 dismissText: string;
 notice: string;
 successTitle: string;
 successDescription: string;
 version: number;
 expectedVersion?: number;
}
export type PublicLeadPopup = Omit<LeadPopupConfig, "version" | "expectedVersion">;
export interface ContactLead {
 id: string;
 contactType: "email" | "whatsapp";
 contact: string;
 firstSubmittedAt: string;
 lastSubmittedAt: string;
 submissionCount: number;
 status: "pending" | "completed" | "invalid";
 note: string;
 version: number;
}
