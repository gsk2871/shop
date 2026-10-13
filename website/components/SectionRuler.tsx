"use client";

import { useEffect, useState } from "react";
import { usePathname } from "next/navigation";

type Chapter = {
  id: string;
  title: string;
  element: HTMLElement;
  widths: number[];
};

const clamp = (value: number, min: number, max: number) =>
  Math.max(min, Math.min(max, value));
const ease = (value: number) => {
  const t = clamp(value, 0, 1);
  return t < 0.5 ? 4 * t ** 3 : 1 - (-2 * t + 2) ** 3 / 2;
};
type Pose = {
  x: number;
  y: number;
  size: number;
  angle: number;
  opacity: number;
};

export default function SectionRuler() {
  const pathname = usePathname();
  const [chapters, setChapters] = useState<Chapter[]>([]);
  const [progress, setProgress] = useState(0);
  const [positions, setPositions] = useState<number[]>([]);
  const [height, setHeight] = useState(720);
  const [documentHeight, setDocumentHeight] = useState(720);
  const [reducedMotion, setReducedMotion] = useState(false);

  useEffect(() => {
    const main = document.querySelector("main");
    if (!main) return;
    let entries: Chapter[] = [];
    let frame = 0;
    let displayed: number[] = [];
    let displayedProgress = 0;
    let lastTime = 0;
    const motion = window.matchMedia("(prefers-reduced-motion: reduce)");
    const update = (time: number) => {
      frame = 0;
      setHeight(window.innerHeight);
      setDocumentHeight(document.documentElement.scrollHeight);
      setReducedMotion(motion.matches);
      const headingPositions = entries.map(
        (entry) => entry.element.getBoundingClientRect().top,
      );
      const max = document.documentElement.scrollHeight - window.innerHeight;
      const position = Math.max(0, window.scrollY);
      const targetProgress =
        max > 0 ? Math.min(100, (position / max) * 100) : 0;
      const follow = motion.matches
        ? 1
        : 1 - Math.exp(-Math.min(64, lastTime ? time - lastTime : 16) / 95);
      lastTime = time;
      let moving = false;
      displayed = entries.map((_, index) => {
        const target = headingPositions[index];
        const previous = displayed[index] ?? target;
        if (Math.abs(target - previous) < 0.1) return target;
        moving = true;
        return previous + (target - previous) * follow;
      });
      if (Math.abs(targetProgress - displayedProgress) > 0.01) moving = true;
      displayedProgress += (targetProgress - displayedProgress) * follow;
      setProgress(displayedProgress);
      setPositions([...displayed]);
      if (moving) frame = window.requestAnimationFrame(update);
      else lastTime = 0;
    };
    const schedule = () => {
      if (!frame) frame = window.requestAnimationFrame(update);
    };
    const discover = () => {
      const context = document.createElement("canvas").getContext("2d");
      if (context) context.font = "600 20px Arial";
      entries = Array.from(main.querySelectorAll<HTMLElement>("h1, h2"))
        .filter(
          (heading) =>
            !heading.closest("a, [hidden], dialog, [role=dialog]") &&
            heading.getClientRects().length > 0 &&
            heading.textContent?.trim(),
        )
        .map((heading, index) => {
          if (!heading.id) heading.id = `reading-chapter-${index + 1}`;
          const title =
            heading.closest<HTMLElement>("[data-ruler-title]")?.dataset
              .rulerTitle || heading.textContent!.trim();
          const text = title.toUpperCase();
          const label = text.length > 25 ? text.slice(0, 24) + "…" : text;
          return {
            id: heading.id,
            title,
            element: heading,
            widths: Array.from(
              label,
              (letter) => context?.measureText(letter).width ?? 12,
            ),
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
    motion.addEventListener("change", schedule);
    window.addEventListener("scroll", schedule, { passive: true });
    window.addEventListener("resize", schedule);
    return () => {
      motion.removeEventListener("change", schedule);
      mutation.disconnect();
      resize.disconnect();
      window.removeEventListener("scroll", schedule);
      window.removeEventListener("resize", schedule);
      if (frame) window.cancelAnimationFrame(frame);
    };
  }, [pathname]);

  if (!chapters.length) return null;
  // Map the document onto the moving ruler, with an inset at both extremes.
  const fraction = progress / 100;
  const scroll = fraction * Math.max(0, documentHeight - height);
  const drift = 60 - fraction * 72;
  const threshold = height * 0.12;
  const models = chapters.map((chapter, index) => {
    const total = chapter.widths.reduce((sum, width) => sum + width, 0);
    const headingSize = Math.min(24, (144 / total) * 20);
    const rideSize = Math.min(20, headingSize);
    const queueSize = Math.min(16, headingSize);
    const y = (positions[index] ?? height) + drift;
    let flip = index === 0 ? 1 : clamp((threshold + 200 - y) / 200, 0, 1);
    // The final short section must still reach its heading slot at page end.
    const anchor = (positions[index] ?? height) + scroll;
    const maxScroll = Math.max(0, documentHeight - height);
    if (maxScroll > 0 && anchor - maxScroll > threshold) {
      const start = Math.max(0, maxScroll - Math.min(200, height * 0.3));
      flip = Math.max(
        flip,
        clamp((scroll - start) / Math.max(1, maxScroll - start), 0, 1),
      );
    }
    return {
      total,
      headingSize,
      rideSize,
      queueSize,
      y,
      flip,
      detach: 1,
      queueY: height - 36,
      rideLength: (total * rideSize) / 20,
    };
  });
  let queued = 0;
  for (let index = models.length - 1; index > 0; index--) {
    const model = models[index];
    model.queueY -= queued * 20;
    model.detach = clamp(
      (model.queueY + 240 - (model.y + model.rideLength)) / 240,
      0,
      1,
    );
    queued += 1 - model.detach;
  }
  const active = Math.max(
    0,
    models.findLastIndex((model) => model.flip >= 1),
  );
  const incoming = clamp(
    models.reduce((sum, model) => sum + (model.flip < 1 ? model.flip : 0), 0),
    0,
    1,
  );
  const layouts = chapters.map((chapter, index) => {
    const model = models[index];
    const text = chapter.title.toUpperCase();
    const label = text.length > 25 ? text.slice(0, 24) + "…" : text;
    let distance = 0;
    const glyphs = Array.from(label, (letter, letterIndex) => {
      const center = distance + chapter.widths[letterIndex] / 2;
      distance += chapter.widths[letterIndex];
      const horizontal = (y: number, size: number, opacity: number): Pose => ({
        x: 156 - ((model.total - center) * size) / 20,
        y,
        size,
        angle: 0,
        opacity,
      });
      const vertical = (y: number): Pose => ({
        x: 176,
        y: y + ((model.total - center) * model.rideSize) / 20,
        size: model.rideSize,
        angle: -90,
        opacity: 0.4,
      });
      let from: Pose;
      let to: Pose;
      let phase: number;
      if (model.flip > 0) {
        const shrink = index === active ? incoming : 1;
        const complete = model.flip >= 1;
        const size = complete
          ? model.headingSize + (model.queueSize - model.headingSize) * shrink
          : model.headingSize;
        const y = 54 + index * 20 + 26 * (complete ? 1 - shrink : 1);
        from = vertical(model.y);
        to = horizontal(y, size, complete ? 1 - 0.6 * shrink : 1);
        phase = model.flip;
      } else if (model.detach < 1) {
        from = horizontal(model.queueY, model.queueSize, 0.4);
        to = vertical(model.queueY - model.rideLength);
        phase = model.detach;
      } else {
        from = to = vertical(model.y);
        phase = 1;
      }
      // The last letter leads both transitions. Each letter travels between
      // its own endpoint poses; the control point bows toward the page.
      const t = reducedMotion
        ? phase >= 0.5
          ? 1
          : 0
        : ease(
            phase * (1 + (label.length - 1) * 0.16) -
              (label.length - 1 - letterIndex) * 0.16,
          );
      const pose = {
        x: (1 - t) * from.x + t * to.x - 56 * t * (1 - t),
        y: (1 - t) * from.y + t * to.y,
        angle: from.angle + (to.angle - from.angle) * t,
        size: from.size + (to.size - from.size) * t,
        opacity: from.opacity + (to.opacity - from.opacity) * t,
      };
      if (reducedMotion) {
        return {
          letter,
          ...horizontal(
            index <= active
              ? 54 + index * 20 + (index === active ? 26 : 0)
              : height - 36 - (chapters.length - 1 - index) * 20,
            index === active ? model.headingSize : model.queueSize,
            index === active ? 1 : 0.4,
          ),
        };
      }
      return { letter, ...pose };
    });
    const left = Math.min(...glyphs.map((glyph) => glyph.x - glyph.size));
    const right = Math.max(...glyphs.map((glyph) => glyph.x + glyph.size));
    const top = Math.min(...glyphs.map((glyph) => glyph.y - glyph.size));
    const bottom = Math.max(...glyphs.map((glyph) => glyph.y + glyph.size));
    return {
      glyphs,
      left,
      top,
      width: right - left,
      height: bottom - top,
      bottom,
    };
  });
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
        <span className="ruler-caption">
          {String(active + 1).padStart(2, "0")} /{" "}
          {String(chapters.length).padStart(2, "0")} · ON THIS PAGE
        </span>
        {chapters.map((chapter, index) => {
          const layout = layouts[index];
          return (
            <button
              key={chapter.id}
              className="ruler-moving-title"
              aria-label={`Jump to ${chapter.title}`}
              aria-current={index === active ? "location" : undefined}
              title={chapter.title}
              onClick={() => jump(chapter)}
              style={{
                top: layout.top,
                left: layout.left,
                width: layout.width,
                height: layout.height,
              }}
            >
              {layout.glyphs.map((glyph, glyphIndex) => (
                <span
                  key={glyphIndex}
                  className="ruler-glyph"
                  aria-hidden="true"
                  style={{
                    left: glyph.x - layout.left,
                    top: glyph.y - layout.top,
                    fontSize: glyph.size,
                    opacity: glyph.opacity,
                    transform: `translate(-50%, -50%) rotate(${glyph.angle}deg)`,
                  }}
                >
                  {glyph.letter === " " ? "\u00a0" : glyph.letter}
                </span>
              ))}
            </button>
          );
        })}
      </nav>
      <div className="ruler-scale" aria-hidden="true">
        {Array.from({ length: 101 }, (_, index) => (
          <span
            key={index}
            className={
              index % 5 === 0 ? "ruler-tick ruler-tick-major" : "ruler-tick"
            }
            style={{
              top: (index / 100) * documentHeight - scroll + drift - 40,
              opacity: layouts.some((layout) => {
                const y = (index / 100) * documentHeight - scroll + drift;
                return y > layout.top - 8 && y < layout.bottom + 8;
              })
                ? 0.15
                : 1,
            }}
          >
            {index % 5 === 0 && <span>{String(index).padStart(2, "0")}</span>}
          </span>
        ))}
        <div
          className="ruler-position"
          style={{ top: 20 + fraction * Math.max(1, height - 72) }}
        >
          <span>{String(Math.round(progress)).padStart(3, "0")}%</span>
        </div>
      </div>
    </aside>
  );
}
