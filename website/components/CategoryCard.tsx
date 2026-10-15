import Link from "next/link";
import type { PublicCategory } from "../lib/contracts";

export default function CategoryCard({
  category,
  index,
}: {
  category: PublicCategory;
  index: number;
}) {
  return (
    <Link
      className="home-category-card"
      href={`/products?categoryId=${encodeURIComponent(category.id)}`}
    >
      {category.cover && (
        <img
          className="home-category-cover"
          src={category.cover}
          alt=""
          loading="lazy"
        />
      )}
      <div className="home-category-copy">
        <span className="eyebrow">{String(index + 1).padStart(2, "0")}</span>
        <h3>{category.name}</h3>
        <span className="home-text-link">Explore products ↗</span>
      </div>
    </Link>
  );
}
