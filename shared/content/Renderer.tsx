import type { ReactNode } from "react";
/** Render only HTML returned by the backend sanitizer, including preview HTML. */
export function RichTextRenderer({
  html,
  mediaPrefix = "",
}: {
  html: string;
  mediaPrefix?: string;
}) {
  const rendered = mediaPrefix
    ? html.replace(
        /src="(\/uploads\/[^"<>]*)"/g,
        (_, path: string) => `src="${mediaPrefix}${path}"`,
      )
    : html;
  return (
    <div
      className="shop-richtext"
      dangerouslySetInnerHTML={{ __html: rendered }}
    />
  );
}
export function ContentFrame({ children }: { children: ReactNode }) {
  return <div className="shop-content-frame">{children}</div>;
}
