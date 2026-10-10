import Link from "next/link";
import Hero from "./Hero";
import { contentResult, banners, settings } from "../lib/content";
import type { Page, PublicArticle } from "../lib/contracts";
export default async function ArticleListing({
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
  const [slides, s] = await Promise.all([banners("research_header"), settings()]);
  const title = s.newsTitle || "News";
  return (
    <>
      <Hero banners={slides} />
      <section className="section">
        <h1>{title}</h1>
        {error ? (
          <p className="empty">
            Articles are temporarily unavailable. Please try again
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
                  href={`/news/${a.slug}`}
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
            <nav className="pagination" aria-label="Article pages">
              {page > 1 && (
                <Link href={`/news?page=${page - 1}`}>← Previous</Link>
              )}
              <span>Page {page}</span>
              {page * data.pageSize < data.total && (
                <Link href={`/news?page=${page + 1}`}>Next →</Link>
              )}
            </nav>
          </>
        )}
      </section>
    </>
  );
}
