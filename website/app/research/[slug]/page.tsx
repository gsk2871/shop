import { permanentRedirect } from "next/navigation";
export default async function ResearchArticleRedirect({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  permanentRedirect(`/news/${encodeURIComponent(slug)}`);
}
