"use client";
import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import type { PublicBanner } from "../lib/contracts";

export default function HomeCarousel({ banners }: { banners: PublicBanner[] }) {
  const [index, setIndex] = useState(0);
  const [hovered, setHovered] = useState(false);
  const [focused, setFocused] = useState(false);
  const [reducedMotion, setReducedMotion] = useState(false);
  const start = useRef<{ x: number; y: number } | null>(null);
  const count = banners.length;
  const active = count ? index % count : 0;
  useEffect(() => {
    const media = window.matchMedia("(prefers-reduced-motion: reduce)");
    const update = () => setReducedMotion(media.matches);
    update(); media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, []);
  useEffect(() => {
    if (count < 2 || hovered || focused || reducedMotion) return;
    const timer = window.setInterval(() => {
      if (!document.hidden) setIndex((i) => (i + 1) % count);
    }, 5000);
    return () => window.clearInterval(timer);
  }, [count, hovered, focused, reducedMotion]);
  const banner = banners[active];
  if (!banner) return null;
  const move = (step: number) => setIndex((i) => (i + step + count) % count);
  return (
    <section className="home-carousel" data-ruler-title="Introduction" aria-label="Featured products" aria-roledescription="carousel"
      onMouseEnter={() => setHovered(true)} onMouseLeave={() => setHovered(false)}
      onFocusCapture={() => setFocused(true)}
      onBlurCapture={(e) => { if (!e.currentTarget.contains(e.relatedTarget)) setFocused(false); }}
      onTouchStart={(e) => { const t = e.touches[0]; start.current = { x: t.clientX, y: t.clientY }; }}
      onTouchCancel={() => { start.current = null; }}
      onTouchEnd={(e) => {
        const t = e.changedTouches[0], origin = start.current; start.current = null;
        if (!origin || count < 2) return;
        const dx = t.clientX - origin.x, dy = t.clientY - origin.y;
        if (Math.abs(dx) > 50 && Math.abs(dx) > Math.abs(dy)) move(dx < 0 ? 1 : -1);
      }}>
      <div className="home-carousel-slide" role="group" aria-roledescription="slide" aria-label={`${active + 1} of ${count}`}>
        <picture>
          {banner.mobileImage && <source media="(max-width: 640px)" srcSet={banner.mobileImage} />}
          <img src={banner.image} alt="" fetchPriority="high" />
        </picture>
        {banner.link && <Link className="button home-carousel-cta" href={banner.link}>Request In-Vitro Quotation ↗</Link>}
      </div>
      {count > 1 && <>
        <button type="button" className="carousel-arrow previous" aria-label="Previous banner" onClick={() => move(-1)}>‹</button>
        <button type="button" className="carousel-arrow next" aria-label="Next banner" onClick={() => move(1)}>›</button>
        <div className="carousel-dots" aria-label="Banner navigation">
          {banners.map((b, i) => <button type="button" key={b.id} aria-label={`Show banner ${i + 1}`} aria-pressed={active === i} onClick={() => setIndex(i)} />)}
        </div>
      </>}
    </section>
  );
}
