import ContactButton from "../../../components/ContactButton";
import { notFound } from "next/navigation";
import { ProductView } from "@shop/content";
import { contentResult } from "../../../lib/content";
import type { PublicProduct } from "../../../lib/contracts";
export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const result = await contentResult<PublicProduct>(
    `products/${encodeURIComponent(slug)}`,
  );
  return {
    title: result.data?.seoTitle || result.data?.name || "Product",
    description: result.data?.seoDescription || result.data?.summary,
    robots: result.error ? { index: false, follow: false } : undefined,
  };
}
export default async function ProductDetail({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const result = await contentResult<PublicProduct>(
    `products/${encodeURIComponent(slug)}`,
  );
  if (result.error?.status === 404) notFound();
  if (!result.data)
    return (
      <section className="section">
        <h1>Product temporarily unavailable.</h1>
        <p>Please try again later.</p>
      </section>
    );
  return (
    <section className="section">
      <ProductView key={result.data.slug} product={result.data} />
      <ContactButton productName={result.data.name}>
        Enquire about this product ↗
      </ContactButton>

    </section>
  );
}
