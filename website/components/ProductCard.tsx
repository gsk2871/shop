import Link from "next/link";
import type { ProductSummary } from "../lib/contracts";
import ContactButton from "./ContactButton";

export default function ProductCard({ product }: { product: ProductSummary }) {
  const href = `/products/${product.slug}`;
  const price = product.fromPrice;
  const recommendation = product.recommendation;
  return <article className="home-product-card product-card">
    <Link href={href} className="product-card-image" aria-label={product.name}>
      {product.image && <img src={product.image} alt={product.name} loading="lazy" decoding="async" />}
      {price?.displayDiscount && <span className="product-saving">Save {price.displayDiscount}</span>}
    </Link>
    <div className="home-card-copy">
      <span className="eyebrow">{product.category}</span>
      <h3><Link href={href}>{product.name}</Link></h3>
      {!!recommendation && recommendation >= 1 && recommendation <= 5 && <p className="product-recommendation" aria-label={`Recommendation: ${recommendation} out of 5`}><span aria-hidden="true">{"★".repeat(recommendation)}<span className="empty-stars">{"☆".repeat(5 - recommendation)}</span></span></p>}
      {product.purityLabel && <p className="product-purity">{product.purityLabel}</p>}
      {price && <p className="product-card-price"><strong>From {price.currency} <span className="product-card-amount">{price.salePrice}</span></strong>{price.displayDiscount && <del>{price.displayOriginalPrice}</del>}</p>}
      <ContactButton className="button marketing-primary product-card-contact" productName={product.name}>Contact us</ContactButton>
      <Link className="home-text-link" href={href}>View specifications ↗</Link>
    </div>
  </article>;
}
