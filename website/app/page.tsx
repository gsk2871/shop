import Link from "next/link";
import Hero from "../components/Hero";
import ContactButton from "../components/ContactButton";
import { settings, banners, content } from "../lib/content";
import type {
  Page,
  ProductSummary,
  PublicCategory,
  PublicArticle,
} from "../lib/contracts";

export default async function Home() {
  const [s, slides, categories, products, news] = await Promise.all([
    settings(),
    banners("home_hero"),
    content<{ items: PublicCategory[] }>("categories"),
    content<Page<ProductSummary>>("products?homeFeatured=true&pageSize=6"),
    content<Page<PublicArticle>>("articles?pageSize=3"),
  ]);
  return (
    <>
      <Hero banners={slides} />
      {!!categories?.items.length && (
        <section
          className="section home-section"
          data-ruler-title="Products"
          aria-labelledby="home-categories"
        >
          <div className="home-section-heading">
            <div>
              <span className="eyebrow">PRODUCT CATEGORIES</span>
              <h2 id="home-categories">Find your starting point.</h2>
            </div>
            <Link className="home-text-link" href="/products">
              All products ↗
            </Link>
          </div>
          <div className="home-category-grid">
            {categories.items.map((category, index) => (
              <Link
                className="home-category-card"
                key={category.id}
                href={`/products?categoryId=${encodeURIComponent(category.id)}`}
              >
                <span className="eyebrow">
                  {String(index + 1).padStart(2, "0")}
                </span>
                <h3>{category.name}</h3>
                <span className="home-text-link">Explore products ↗</span>
              </Link>
            ))}
          </div>
        </section>
      )}
      {!!products?.items.length && (
        <section
          className="section home-section home-featured"
          data-ruler-title="Featured products"
          aria-labelledby="home-products"
        >
          <div className="home-section-heading">
            <div>
              <span className="eyebrow">SELECTED FROM OUR CATALOG</span>
              <h2 id="home-products">Featured products.</h2>
            </div>
            <Link className="home-text-link" href="/products">
              View all products ↗
            </Link>
          </div>
          <div className="home-card-grid">
            {products.items.map((product) => (
              <Link
                className="home-product-card"
                key={product.id}
                href={`/products/${product.slug}`}
              >
                {product.image && (
                  <img
                    className="home-product-image"
                    src={product.image}
                    alt={product.name}
                    loading="lazy"
                  />
                )}
                <div className="home-card-copy">
                  <span className="eyebrow">{product.category}</span>
                  <h3>{product.name}</h3>
                  <p>{product.summary}</p>
                  {product.fromPrice && (
                    <p className="home-price">
                      From {product.fromPrice.displayPrice}
                    </p>
                  )}
                  <span className="home-text-link">View specifications ↗</span>
                </div>
              </Link>
            ))}
          </div>
        </section>
      )}
      {(s.homeTitle || s.homeContent) && (
        <section
          className="section home-section home-about"
          data-ruler-title="About us"
          aria-labelledby="home-about"
        >
          <div>
            <span className="eyebrow">GET TO KNOW US</span>
            <h2 id="home-about">{s.homeTitle || "About us"}</h2>
          </div>
          <div>
            <div className="configured-copy">{s.homeContent}</div>
            <Link className="home-text-link" href="/about">
              More about us ↗
            </Link>
          </div>
        </section>
      )}
      {!!news?.items.length && (
        <section
          className="section home-section"
          data-ruler-title="News"
          aria-labelledby="home-news"
        >
          <div className="home-section-heading">
            <div>
              <span className="eyebrow">NEWS & UPDATES</span>
              <h2 id="home-news">From our journal.</h2>
            </div>
            <Link className="home-text-link" href="/news">
              All news ↗
            </Link>
          </div>
          <div className="home-card-grid">
            {news.items.map((article) => (
              <Link
                className="home-news-card"
                key={article.id}
                href={`/news/${article.slug}`}
              >
                {article.cover && (
                  <img
                    className="home-news-image"
                    src={article.cover}
                    alt=""
                    loading="lazy"
                  />
                )}
                <div className="home-card-copy">
                  {article.firstPublishedAt && (
                    <time
                      className="eyebrow"
                      dateTime={article.firstPublishedAt}
                    >
                      {article.firstPublishedAt.slice(0, 10)}
                    </time>
                  )}
                  <h3>{article.title}</h3>
                  <p>{article.summary}</p>
                  <span className="home-text-link">Read article ↗</span>
                </div>
              </Link>
            ))}
          </div>
        </section>
      )}
      <section
        className="section home-section home-contact"
        data-ruler-title="Get in touch"
        aria-labelledby="home-contact"
      >
        <span className="eyebrow">LET’S TALK</span>
        <h2 id="home-contact">Find the right products for your project.</h2>
        <p>
          Contact our team to discuss product specifications, quantities,
          pricing and delivery arrangements.
        </p>
        <div className="actions">
          <ContactButton>Contact our team ↗</ContactButton>
          <Link className="home-text-link" href="/faq">
            Visit FAQ ↗
          </Link>
        </div>
      </section>
    </>
  );
}
