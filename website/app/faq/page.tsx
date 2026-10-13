import type { Metadata } from "next";
import { contentResult } from "../../lib/content";
import type { PublicFAQ } from "../../lib/contracts";

export const metadata: Metadata = { title: "FAQ" };

export default async function FAQ() {
  const { data, error } = await contentResult<{ items: PublicFAQ[] }>("faqs");
  return (
    <section className="section faq-section" aria-labelledby="faq-title">
      <h1 id="faq-title">Frequently asked questions</h1>
      {error ? (
        <p>Questions are temporarily unavailable. Please try again later.</p>
      ) : data?.items.length ? (
        <div className="faq-list">
          {data.items.map((faq) => (
            <details key={faq.id} className="faq-item">
              <summary>{faq.question}</summary>
              <div className="faq-answer">{faq.answer}</div>
            </details>
          ))}
        </div>
      ) : (
        <p>No questions have been published yet.</p>
      )}
    </section>
  );
}
