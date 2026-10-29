"use client";
import { useEffect, useRef, useState, type CSSProperties } from "react";
import type { Announcement } from "../lib/contracts";

export default function AnnouncementBar({ config }: { config?: Announcement | null }) {
  const viewport = useRef<HTMLDivElement>(null);
  const group = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);
  const items = config?.enabled ? (config.items ?? []).filter(item => item.enabled) : [];
  const contentKey = items.map(item => item.text).join("\n");
  useEffect(() => {
    const view = viewport.current, content = group.current;
    if (!view || !content) return;
    const measure = () => setWidth(content.scrollWidth > view.clientWidth ? content.scrollWidth : 0);
    const observer = new ResizeObserver(measure);
    observer.observe(view); observer.observe(content); measure();
    return () => observer.disconnect();
  }, [contentKey]);
  if (!config || !items.length) return null;
  const speed = { slow: 30, standard: 50, fast: 80 }[config.speed] || 50;
  const style = { backgroundColor: config.background, color: config.textColor, "--announcement-duration": `${Math.max(width / speed, 1)}s` } as CSSProperties;
  const text = items.map((item, i) => <span key={i}>{item.text}</span>);
  return <div ref={viewport} className={`announcement-bar${width ? " is-scrolling" : ""}`} style={style} role="region" aria-label="Announcements" tabIndex={width ? 0 : undefined}>
    <div className="announcement-track">
      <div ref={group} className="announcement-group">{text}</div>
      {!!width && <div className="announcement-group announcement-copy" aria-hidden="true">{text}</div>}
    </div>
  </div>;
}
