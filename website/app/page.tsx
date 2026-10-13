import Hero from "../components/Hero";
import { settings, banners, content } from "../lib/content";
import type { PublicFAQ } from "../lib/contracts";
export default async function Home() {
  const [s, slides, faqs] = await Promise.all([
    settings(),
    banners("home_hero"),
    content<{ items: PublicFAQ[] }>("faqs"),
  ]);
  return (
    <>
      <Hero banners={slides} />
      {(s.homeTitle || s.homeContent) && (
        <section className="section prose">
          {s.homeTitle && <h1>{s.homeTitle}</h1>}
          <div className="configured-copy">{s.homeContent}</div>
        </section>
      )}
      {!!faqs?.items.length && (
        <section className="section faq-section" aria-labelledby="faq-title">
          <h2 id="faq-title">Frequently asked questions</h2>
          <div className="faq-list">
            {faqs.items.map((faq) => (
              <details key={faq.id} className="faq-item">
                <summary>{faq.question}</summary>
                <div className="faq-answer">{faq.answer}</div>
              </details>
            ))}
          </div>
        </section>
      )}
    </>
  );
}
