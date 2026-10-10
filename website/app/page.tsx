import Hero from "../components/Hero";
import { settings, banners } from "../lib/content";
export default async function Home() {
  const [s, slides] = await Promise.all([settings(), banners("home_hero")]);
  return (
    <>
      <Hero banners={slides} />
      {(s.homeTitle || s.homeContent) && (
        <section className="section prose">
          {s.homeTitle && <h1>{s.homeTitle}</h1>}
          <div className="configured-copy">{s.homeContent}</div>
        </section>
      )}
    </>
  );
}
