"use client";

import { useEffect, useState } from "react";
import { usePathname } from "next/navigation";

type Chapter = { id: string; title: string; element: HTMLElement };

export default function SectionRuler() {
  const pathname = usePathname();
  const [chapters, setChapters] = useState<Chapter[]>([]);
  const [active, setActive] = useState(0);
  const [progress, setProgress] = useState(0);

  useEffect(() => {
    const main = document.querySelector("main");
    if (!main) return;
    let entries: Chapter[] = [];
    let frame = 0;
    const update = () => {
      frame = 0;
      const max = document.documentElement.scrollHeight - window.innerHeight;
      const position = Math.max(0, window.scrollY);
      setProgress(max > 0 ? Math.min(100, (position / max) * 100) : 0);
      // A chapter becomes current when its heading reaches the upper reading area.
      const readingLine = Math.min(180, window.innerHeight * 0.25);
      let current = 0;
      entries.forEach((entry, index) => {
        if (entry.element.getBoundingClientRect().top <= readingLine)
          current = index;
      });
      if (max > 0 && position >= max - 2)
        current = Math.max(0, entries.length - 1);
      setActive(current);
    };
    const schedule = () => {
      if (!frame) frame = window.requestAnimationFrame(update);
    };
    const discover = () => {
      entries = Array.from(main.querySelectorAll<HTMLElement>("h1, h2"))
        .filter(
          (heading) =>
            !heading.closest("a, [hidden], dialog, [role=dialog]") &&
            heading.getClientRects().length > 0 &&
            heading.textContent?.trim(),
        )
        .map((heading, index) => {
          if (!heading.id) heading.id = `reading-chapter-${index + 1}`;
          return {
            id: heading.id,
            title:
              heading.closest<HTMLElement>("[data-ruler-title]")?.dataset
                .rulerTitle || heading.textContent!.trim(),
            element: heading,
          };
        });
      setChapters(entries);
      schedule();
    };
    discover();
    const mutation = new MutationObserver(discover);
    mutation.observe(main, {
      childList: true,
      subtree: true,
      characterData: true,
    });
    const resize = new ResizeObserver(schedule);
    resize.observe(document.body);
    window.addEventListener("scroll", schedule, { passive: true });
    window.addEventListener("resize", schedule);
    return () => {
      mutation.disconnect();
      resize.disconnect();
      window.removeEventListener("scroll", schedule);
      window.removeEventListener("resize", schedule);
      if (frame) window.cancelAnimationFrame(frame);
    };
  }, [pathname]);

  const current = chapters[active];
  if (!current) return null;
  const previous = chapters[active - 1];
  const next = chapters[active + 1];
  const jump = (chapter: Chapter) => {
    chapter.element.scrollIntoView({
      behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches
        ? "instant"
        : "smooth",
      block: "start",
    });
    chapter.element.setAttribute("tabindex", "-1");
    chapter.element.focus({ preventScroll: true });
  };
  return (
    <aside className="section-ruler" aria-label="Page reading guide">
      <nav className="ruler-chapters" aria-label="Page sections">
        <div className="ruler-current">
          <span className="ruler-caption">ON THIS PAGE</span>
          {previous && (
            <button className="ruler-previous" onClick={() => jump(previous)}>
              ↑ {previous.title}
            </button>
          )}
          <span className="ruler-index">
            {String(active + 1).padStart(2, "0")} /{" "}
            {String(chapters.length).padStart(2, "0")}
          </span>
          <button
            className="ruler-title"
            aria-current="location"
            onClick={() => jump(current)}
          >
            {current.title}
          </button>
        </div>
        {next ? (
          <button className="ruler-next" onClick={() => jump(next)}>
            <span className="ruler-caption">NEXT SECTION ↓</span>
            <span>{next.title}</span>
          </button>
        ) : (
          <button className="ruler-next" onClick={() => jump(chapters[0])}>
            <span className="ruler-caption">BACK TO START ↑</span>
            <span>{chapters[0].title}</span>
          </button>
        )}
      </nav>
      <div className="ruler-scale" aria-hidden="true">
        {Array.from({ length: 51 }, (_, index) => (
          <span
            key={index}
            className={
              index % 5 === 0 ? "ruler-tick ruler-tick-major" : "ruler-tick"
            }
            style={{ top: `${index * 2}%` }}
          >
            {index % 5 === 0 && (
              <span>{String(index * 2).padStart(2, "0")}</span>
            )}
          </span>
        ))}
        <div className="ruler-position" style={{ top: `${progress}%` }}>
          <span>{String(Math.round(progress)).padStart(3, "0")}%</span>
        </div>
      </div>
    </aside>
  );
}
