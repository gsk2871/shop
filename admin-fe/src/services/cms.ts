export interface Banner {
	title: string;
	subtitle: string;
	image: string;
	link: string;
}
export interface Settings {
	brand: string;
	logo: string;
	primaryColor: string;
	backgroundColor: string;
	description: string;
	email: string;
	whatsapp: string;
	disclaimer: string;
	banners: Banner[];
}
export interface Article {
	id: number;
	title: string;
	slug: string;
	summary: string;
	body: string;
	cover: string;
	status: string;
	seoTitle: string;
	seoDescription: string;
	updatedAt: string;
}
export interface Media {
	id: number;
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
	if (!response.ok) throw new Error(data.error || "请求失败");
	return data;
}
export function assetURL(url: string) {
	return url.startsWith("/uploads/") ? `/backend${url}` : url;
}

export interface Product {
 id: number;
 name: string;
 slug: string;
 category: string;
 aliases: string;
 cas: string;
 formula: string;
 molecularWeight: string;
 sequence: string;
 purity: string;
 specifications: string;
 storage: string;
 solubility: string;
 summary: string;
 description: string;
 images: string[];
 status: string;
 seoTitle: string;
 seoDescription: string;
 updatedAt: string;
}
