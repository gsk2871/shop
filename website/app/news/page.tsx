import ArticleListing from "../../components/ArticleListing";
import { settings } from "../../lib/content";
export async function generateMetadata() {
  const s = await settings();
  return { title: s.newsTitle || "News" };
}
export default function News({ searchParams }: { searchParams: Promise<{ page?: string }> }) {
  return <ArticleListing searchParams={searchParams} />;
}
