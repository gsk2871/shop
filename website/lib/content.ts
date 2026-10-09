import "server-only";
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
	seoTitle: string;
	seoDescription: string;
	updatedAt: string;
}
export const defaults: Settings = {
	brand: "Research Supply",
	logo: "",
	primaryColor: "#174e42",
	backgroundColor: "#f6f5ef",
	description: "Materials and documentation for laboratory research.",
	email: "",
	whatsapp: "",
	disclaimer:
		"For laboratory and research use only. Not for human or clinical consumption.",
	banners: [
		{
			title: "Research starts with the right materials.",
			subtitle:
				"Explore our laboratory research catalogue and request documentation.",
			image: "",
			link: "/contact",
		},
	],
};
export async function content<T>(path: string): Promise<T | null> {
	try {
		const r = await fetch(
			`${process.env.BACKEND_URL || "http://127.0.0.1:8080"}/api/v1/${path}`,
			{ cache: "no-store", signal: AbortSignal.timeout(3000) },
		);
		if (!r.ok) return null;
		return await r.json();
	} catch {
		return null;
	}
}
export async function settings() {
	return (await content<Settings>("site")) || defaults;
}
