import Link from "next/link";
import Hero from "../../components/Hero";
import { contentResult, banners } from "../../lib/content";
import type { Page, PublicArticle } from "../../lib/contracts";
export const metadata = { title: "Research library" };
export default async function Research({
  searchParams,
}: {
  searchParams: Promise<{ page?: string }>;
}) {
  const params = await searchParams;
  const page = Math.max(
    1,
    Math.min(10000000, Number.parseInt(params.page || "1", 10) || 1),
  );
  const { data, error } = await contentResult<Page<PublicArticle>>(
    `articles?page=${page}&pageSize=20`,
  );
  const slides = await banners("research_header");
  return (
    <>
      <Hero banners={slides} />
      <section className="section">
        <span className="eyebrow">KNOWLEDGE & REFERENCE</span>
        <h1>Research library.</h1>
        {error ? (
          <p className="empty">
            The research library is temporarily unavailable. Please try again
            later.
          </p>
        ) : !data?.items.length ? (
          <p className="empty">No articles have been published yet.</p>
        ) : (
          <>
            <div className="article-grid">
              {data.items.map((a) => (
                <Link
                  className="article-card"
                  href={`/research/${a.slug}`}
                  key={a.id}
                >
                  {a.cover && <img loading="lazy" src={a.cover} alt="" />}
                  <h2>{a.title}</h2>
                  <p>{a.summary}</p>
                  <time>{a.firstPublishedAt?.slice(0, 10)}</time>
                  <span>Read article →</span>
                </Link>
              ))}
            </div>
            <nav className="pagination" aria-label="Research pages">
              {page > 1 && (
                <Link href={`/research?page=${page - 1}`}>← Previous</Link>
              )}
              <span>Page {page}</span>
              {page * data.pageSize < data.total && (
                <Link href={`/research?page=${page + 1}`}>Next →</Link>
              )}
            </nav>
          </>
        )}
      </section>
    </>
  );
}
