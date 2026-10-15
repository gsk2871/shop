"use client";
import { useState, useMemo, type ReactNode } from "react";
import type { PublicProduct } from "@shop/contracts";
import { RichTextRenderer } from "./Renderer";
/** Both website details and unsaved admin preview render this same component. */
export function ProductView({
  product,
  mediaPrefix = "",
  contact,
}: {
  product: PublicProduct;
  mediaPrefix?: string;
  contact?: ReactNode;
}) {
  const images = product.images ?? (product.image ? [product.image] : []);
  const reports = product.reports ?? (product.report ? [product.report] : []);
  const [activeImage, setActiveImage] = useState(0);
  const imageURL = (url: string) =>
    url.startsWith("/uploads/") ? mediaPrefix + url : url;
  const [selected, setSelected] = useState<Record<string, string>>({});
  const valid = useMemo(
    () => product.skus.filter((sku) => sku.price),
    [product.skus],
  );
  const chosen = valid.find((sku) =>
    product.specs.every(
      (spec) =>
        selected[spec.key] && sku.specValueKeys.includes(selected[spec.key]),
    ),
  );
  const price = product.specs.length === 0 ? valid[0]?.price : chosen?.price;
  function available(spec: string, value: string) {
    return valid.some(
      (sku) =>
        sku.specValueKeys.includes(value) &&
        product.specs.every(
          (dimension) =>
            dimension.key === spec ||
            !selected[dimension.key] ||
            sku.specValueKeys.includes(selected[dimension.key]),
        ),
    );
  }
  return (
    <article className="shop-product-view">
      <h1>{product.name}</h1>
      <div className="shop-product-overview">
        <div className="shop-product-gallery">
          {images.length > 0 && (
            <img
              className="shop-product-main-image"
              src={imageURL(images[activeImage] || images[0])}
              alt={`${product.name} — image ${activeImage + 1}`}
            />
          )}
          {images.length > 1 && (
            <div
              className="shop-product-thumbnails"
              aria-label="Product images"
            >
              {images.map((url, index) => (
                <button
                  type="button"
                  key={`${url}-${index}`}
                  aria-label={`View product image ${index + 1}`}
                  aria-pressed={activeImage === index}
                  onClick={() => setActiveImage(index)}
                >
                  <img src={imageURL(url)} alt="" />
                </button>
              ))}
            </div>
          )}
        </div>
        <div className="shop-product-info">
          <p className="shop-product-summary">{product.summary}</p>
          {product.category && (
            <dl className="shop-product-facts">
              <div>
                <dt>Category</dt>
                <dd>{product.category}</dd>
              </div>
            </dl>
          )}
          {product.specs.map((spec) => (
            <fieldset className="shop-spec" key={spec.key}>
              <legend>{spec.name}</legend>
              {spec.values.map((value) => (
                <button
                  key={value.key}
                  type="button"
                  disabled={!available(spec.key, value.key)}
                  aria-pressed={selected[spec.key] === value.key}
                  onClick={() =>
                    setSelected((current) => ({
                      ...current,
                      [spec.key]:
                        current[spec.key] === value.key ? "" : value.key,
                    }))
                  }
                >
                  {value.name}
                </button>
              ))}
            </fieldset>
          ))}
          <p className="shop-price">
            {price?.displayPrice ||
              (product.specs.length && product.fromPrice
                ? `From ${product.fromPrice.displayPrice} · Select specifications`
                : "Price not available")}
          </p>
          <div className="shop-product-contact">
            {contact || (
              <button type="button" disabled>
                Enquire about this product ↗
              </button>
            )}
          </div>
        </div>
      </div>
      {reports.length > 0 && (
        <section className="shop-product-reports">
          <h2>Test report</h2>
          <div className="shop-product-report-list">
            {reports.map((url, index) => (
              <a
                key={`${url}-${index}`}
                href={imageURL(url)}
                target="_blank"
                rel="noreferrer"
                aria-label={`Open test report ${index + 1}`}
              >
                <img
                  className="shop-product-report"
                  src={imageURL(url)}
                  alt={`${product.name} test report ${index + 1}`}
                  loading="lazy"
                />
              </a>
            ))}
          </div>
        </section>
      )}
      {product.detail.html && (
        <section className="shop-product-description">
          <h2>Description</h2>
          <RichTextRenderer
            html={product.detail.html}
            mediaPrefix={mediaPrefix}
          />
        </section>
      )}
    </article>
  );
}
