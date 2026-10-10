import type { PublicArticle } from "@shop/contracts";
import { RichTextRenderer } from "./Renderer";
export function ArticleView({
  article: a,
  mediaPrefix = "",
}: {
  article: PublicArticle;
  mediaPrefix?: string;
}) {
  return (
    <article className="shop-article">
      <p className="eyebrow">SCIENTIFIC RESEARCH REFERENCE</p>
      <h1>{a.title}</h1>
      <p>{a.summary}</p>
      {a.firstPublishedAt && (
        <time dateTime={a.firstPublishedAt}>
          {a.firstPublishedAt.slice(0, 10)}
        </time>
      )}
      {a.cover && (
        <img
          className="cover"
          src={
            a.cover.startsWith("/uploads/") ? mediaPrefix + a.cover : a.cover
          }
          alt=""
        />
      )}
      <RichTextRenderer html={a.body.html} mediaPrefix={mediaPrefix} />
      <aside>
        This article is for scientific research reference only. Compounds
        discussed are strictly for laboratory analysis.
      </aside>
    </article>
  );
}
