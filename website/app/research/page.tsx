import { permanentRedirect } from "next/navigation";
export default async function ResearchRedirect({
  searchParams,
}: {
  searchParams: Promise<{ page?: string }>;
}) {
  const { page } = await searchParams;
  permanentRedirect(page ? `/news?page=${encodeURIComponent(page)}` : "/news");
}
