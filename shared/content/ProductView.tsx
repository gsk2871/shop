"use client";
import { useState, useMemo } from "react";
import type { PublicProduct } from "@shop/contracts";
import { RichTextRenderer } from "./Renderer";
/** Both website details and unsaved admin preview render this same component. */
export function ProductView({
  product,
  mediaPrefix = "",
}: {
  product: PublicProduct;
  mediaPrefix?: string;
}) {
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
      <p>{product.summary}</p>
      <dl className="shop-product-facts">
        {[
          ["Category", product.category],
          ["Aliases", product.aliases],
          ["CAS", product.cas],
          ["Formula", product.formula],
          ["Molecular weight", product.molecularWeight],
          ["Sequence", product.sequence],
          ["Purity", product.purity],
          ["Storage", product.storage],
          ["Solubility", product.solubility],
        ]
          .filter(([, value]) => value)
          .map(([label, value]) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>{value}</dd>
            </div>
          ))}
      </dl>
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
                  [spec.key]: current[spec.key] === value.key ? "" : value.key,
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
      <RichTextRenderer html={product.detail.html} mediaPrefix={mediaPrefix} />
    </article>
  );
}
