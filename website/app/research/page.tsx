import Link from "next/link";
import { content, type Article } from "../../lib/content";
export const metadata = { title: "Research library" };
export default async function Research() {
	const rows = await content<Article[]>("articles");
	return (
		<section className="section">
			<span className="eyebrow">KNOWLEDGE & REFERENCE</span>
			<h1>Research library.</h1>
			<div className="article-grid">
				{rows?.length ? (
					rows.map((a) => (
						<Link
							className="article-card"
							href={`/research/${a.slug}`}
							key={a.id}
						>
							{a.cover && <img src={a.cover} alt="" />}
							<h2>{a.title}</h2>
							<p>{a.summary}</p>
							<span>Read article →</span>
						</Link>
					))
				) : (
					<p className="empty">
						{rows === null
							? "The research library is temporarily unavailable. Please try again later."
							: "No articles have been published yet."}
					</p>
				)}
			</div>
		</section>
	);
}
