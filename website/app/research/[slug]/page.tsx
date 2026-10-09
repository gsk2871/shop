import { notFound } from "next/navigation";
import { content, type Article } from "../../../lib/content";
export async function generateMetadata({
	params,
}: {
	params: Promise<{ slug: string }>;
}) {
	const { slug } = await params;
	const a = await content<Article>(`articles/${encodeURIComponent(slug)}`);
	return {
		title: a?.seoTitle || a?.title || "Article",
		description: a?.seoDescription || a?.summary,
	};
}
export default async function ArticlePage({
	params,
}: {
	params: Promise<{ slug: string }>;
}) {
	const { slug } = await params;
	const a = await content<Article>(`articles/${encodeURIComponent(slug)}`);
	if (!a) notFound();
	return (
		<article className="section prose">
			<span className="eyebrow">SCIENTIFIC RESEARCH REFERENCE</span>
			<h1>{a.title}</h1>
			<p className="intro">{a.summary}</p>
			{a.cover && <img className="cover" src={a.cover} alt="" />}
			{a.body.split(/\n\s*\n/).map((p, i) => (
				<p key={i} style={{ whiteSpace: "pre-wrap" }}>
					{p}
				</p>
			))}
			<aside className="article-disclaimer">
				This article is for scientific research reference only. Compounds
				discussed are strictly for laboratory analysis.
			</aside>
		</article>
	);
}
