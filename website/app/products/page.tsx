import Link from "next/link";
import Hero from "../../components/Hero";
import { contentResult, banners, settings } from "../../lib/content";
import type { Page, ProductSummary, PublicCategory } from "../../lib/contracts";
export async function generateMetadata() {
  const s = await settings();
  return { title: s.productsTitle || "Products" };
}
export default async function Products({
  searchParams,
}: {
  searchParams: Promise<{
    q?: string | string[];
    page?: string | string[];
    categoryId?: string | string[];
  }>;
}) {
  const params = await searchParams;
  const single = (value: string | string[] | undefined) =>
    Array.isArray(value) ? value[0] || "" : value || "";
  const page = Math.max(
    1,
    Math.min(10000000, Number.parseInt(single(params.page) || "1", 10) || 1),
  );
  const query = single(params.q).trim();
  const categoryId = single(params.categoryId);
  const url = (category: string, nextPage = 1) => {
    const qs = new URLSearchParams();
    if (query) qs.set("q", query);
    if (category) qs.set("categoryId", category);
    if (nextPage > 1) qs.set("page", String(nextPage));
    return `/products${qs.size ? `?${qs}` : ""}`;
  };
  const listParams = new URLSearchParams({
    page: String(page),
    pageSize: "20",
  });
  if (query) listParams.set("q", query);
  if (categoryId) listParams.set("categoryId", categoryId);
  const [result, categoryResult, slides, s] = await Promise.all([
    contentResult<Page<ProductSummary>>(`products?${listParams}`),
    contentResult<{ items: PublicCategory[] }>("categories"),
    banners("products_header"),
    settings(),
  ]);
  const data = result.data;
  return (
    <>
      <Hero banners={slides} />
      <section className="section">
        <div className="product-finder">
          <div className="product-finder-heading">
            <h1>{s.productsTitle || "Products"}</h1>
            {data && (
              <span>
                <strong>{data.total}</strong> products
              </span>
            )}
          </div>
          <form action="/products" className="product-search">
            <label htmlFor="product-search" className="sr-only">
              Search by product name
            </label>
            <svg
              width="24"
              height="24"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              aria-hidden="true"
            >
              <circle cx="10" cy="10" r="7" />
              <path d="m15 15 6 6" />
            </svg>
            <input
              id="product-search"
              name="q"
              defaultValue={query}
              placeholder="Product name…"
              maxLength={200}
            />
            {categoryId && (
              <input type="hidden" name="categoryId" value={categoryId} />
            )}
            <button
              type="submit"
              className="button"
              aria-label="Search products"
            >
              →
            </button>
          </form>
          <nav className="product-categories" aria-label="Product categories">
            <Link
              href={url("")}
              aria-current={!categoryId ? "page" : undefined}
            >
              All products
            </Link>
            {categoryResult.data?.items.map((category) => (
              <Link
                key={category.id}
                href={url(category.id)}
                aria-current={categoryId === category.id ? "page" : undefined}
              >
                {category.name}
              </Link>
            ))}
          </nav>
          {categoryResult.error && (
            <p role="status">
              Categories are temporarily unavailable. Please try again later.
            </p>
          )}
        </div>
        {result.error ? (
          <p className="empty">
            The catalogue is temporarily unavailable. Please try again later.
          </p>
        ) : !data?.items.length ? (
          <p className="empty">No products match your search.</p>
        ) : (
          <>
            <div className="article-grid product-results">
              {data.items.map((product) => (
                <Link
                  key={product.id}
                  className="article-card"
                  href={`/products/${product.slug}`}
                >
                  <span className="eyebrow">{product.category}</span>
                  <h2>{product.name}</h2>
                  <p>{product.summary}</p>
                  <p>
                    {product.fromPrice
                      ? `From ${product.fromPrice.displayPrice}`
                      : ""}
                  </p>
                  <span>View specifications →</span>
                </Link>
              ))}
            </div>
            <nav className="pagination" aria-label="Product pages">
              {page > 1 && (
                <Link href={url(categoryId, page - 1)}>← Previous</Link>
              )}
              <span>Page {page}</span>
              {page * data.pageSize < data.total && (
                <Link href={url(categoryId, page + 1)}>Next →</Link>
              )}
            </nav>
          </>
        )}
      </section>
    </>
  );
}
