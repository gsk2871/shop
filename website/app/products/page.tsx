import Link from "next/link";
import Hero from "../../components/Hero";
import { contentResult, banners, settings } from "../../lib/content";
import type { Page, ProductSummary } from "../../lib/contracts";
export async function generateMetadata() {
  const s = await settings();
  return { title: s.productsTitle || "Products" };
}
export default async function Products({
  searchParams,
}: {
  searchParams: Promise<{ q?: string; page?: string }>;
}) {
  const params = await searchParams;
  const page = Math.max(
    1,
    Math.min(10000000, Number.parseInt(params.page || "1", 10) || 1),
  );
  const query = params.q || "";
  const result = await contentResult<Page<ProductSummary>>(
    `products?page=${page}&pageSize=20&q=${encodeURIComponent(query)}`,
  );
  const data = result.data;
  const [slides, s] = await Promise.all([banners("products_header"), settings()]);
  return (
    <>
      <Hero banners={slides} />
      <section className="section">
        <h1>{s.productsTitle || "Products"}</h1>
        <form action="/products" className="catalogue-search">
          <label htmlFor="product-search">Search products</label>
          <input
            id="product-search"
            name="q"
            defaultValue={query}
            placeholder="Name, CAS or category"
            maxLength={200}
          />
          <button type="submit" className="button">
            Search
          </button>
        </form>
        {result.error ? (
          <p className="empty">
            The catalogue is temporarily unavailable. Please try again later.
          </p>
        ) : !data?.items.length ? (
          <p className="empty">No products match your search.</p>
        ) : (
          <>
            <div className="article-grid">
              {data.items.map((product) => (
                <Link
                  key={product.id}
                  className="article-card"
                  href={`/products/${product.slug}`}
                >
                  <span className="eyebrow">
                    {product.category}
                  </span>
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
                <Link
                  href={`/products?page=${page - 1}&q=${encodeURIComponent(query)}`}
                >
                  ← Previous
                </Link>
              )}
              <span>Page {page}</span>
              {page * data.pageSize < data.total && (
                <Link
                  href={`/products?page=${page + 1}&q=${encodeURIComponent(query)}`}
                >
                  Next →
                </Link>
              )}
            </nav>
          </>
        )}
      </section>
    </>
  );
}
