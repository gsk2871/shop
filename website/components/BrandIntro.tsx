import type { BrandIntro as Config } from "../lib/contracts";
import MarketingActions from "./MarketingActions";

export default function BrandIntro({ config }: { config?: Config | null }) {
  if (!config) return null;
  const parts = config.parts ?? [], metrics = config.metrics ?? [];
  if (!config.tagline && !parts.length && !config.description && !metrics.length) return null;
  return <section className="section brand-intro" data-ruler-title="Our brand" aria-label="Our brand">
    {config.tagline && <p className="brand-intro-tagline">{config.tagline}</p>}
    {!!parts.length && <h2 className="brand-intro-name">{parts.map((part, i) => <span key={i} style={{ color: part.color }}>{part.text}</span>)}</h2>}
    {config.description && <p className="brand-intro-description">{config.description}</p>}
    {!!metrics.length && <dl className="brand-metrics">{metrics.map((metric, i) => <div key={i}><dt>{metric.description}</dt><dd>{metric.value}</dd></div>)}</dl>}
    <MarketingActions />
  </section>;
}
