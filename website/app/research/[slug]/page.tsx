import { notFound } from "next/navigation";
import { ArticleView } from "@shop/content";
import { contentResult } from "../../../lib/content";
import type { PublicArticle } from "../../../lib/contracts";
export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const result = await contentResult<PublicArticle>(
    `articles/${encodeURIComponent(slug)}`,
  );
  return {
    title: result.data?.seoTitle || result.data?.title || "Article",
    description: result.data?.seoDescription || result.data?.summary,
    robots: result.error ? { index: false, follow: false } : undefined,
  };
}
export default async function ArticlePage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const result = await contentResult<PublicArticle>(
    `articles/${encodeURIComponent(slug)}`,
  );
  if (result.error?.status === 404) notFound();
  if (!result.data)
    return (
      <section className="section">
        <h1>Article temporarily unavailable.</h1>
        <p>Please try again later.</p>
      </section>
    );
  return (
    <section className="section">
      <ArticleView article={result.data} />
    </section>
  );
}
