import ContactButton from "../components/ContactButton";
import Link from "next/link";
import Hero from "../components/Hero";
import { settings, content, type Article, banners } from "../lib/content";
export default async function Home() {
  const [s, articlePage, slides] = await Promise.all([
    settings(),
    content<{ items: Article[] }>("articles?pageSize=3"),
    banners("home_hero"),
  ]);
  const articles = articlePage?.items || (articlePage === null ? null : []);
  return (
    <>
      <Hero
        banners={
          slides.length
            ? slides
            : [
                {
                  id: "default",
                  title: "Research starts with the right materials.",
                  subtitle:
                    "Explore our laboratory research catalogue and request documentation.",
                  image: "",
                  mobileImage: "",
                  ctaText: "Make an enquiry",
                  link: "",
                },
              ]
        }
      />
      <div className="principles">
        <span>01 / Research use only</span>
        <span>02 / Documentation enquiries</span>
        <span>03 / Direct contact</span>
      </div>
      <section className="section split">
        <div>
          <span className="eyebrow">THE RESEARCH COLLECTION</span>
          <h2>
            A considered approach
            <br />
            to laboratory supply.
          </h2>
        </div>
        <div>
          <p>{s.description}</p>
          <p>
            Discuss your material requirements, request available documentation
            and confirm specifications directly with our team.
          </p>
          <ContactButton className="text-link">
            Discuss your requirements →
          </ContactButton>
        </div>
      </section>
      <section className="documentation">
        <div>
          <span className="eyebrow">DOCUMENTATION FIRST</span>
          <h2>
            Know your materials.
            <br />
            Ask for the details.
          </h2>
          <p>
            Request available product information and testing documentation
            before confirming your enquiry.
          </p>
          <Link className="button light" href="/reports">
            View documentation →
          </Link>
        </div>
        <div className="document-card">
          <div className="document-top">
            RESEARCH DOCUMENTATION<span>↗</span>
          </div>
          <h3>Clarity at every step.</h3>
          {[
            "Product identification",
            "Available testing reports",
            "Specification confirmation",
          ].map((v, i) => (
            <div className="document-row" key={v}>
              <span>0{i + 1}</span>
              {v}
            </div>
          ))}
          <small>Documentation availability varies by product.</small>
        </div>
      </section>
      <section className="section">
        <div className="section-heading">
          <div>
            <span className="eyebrow">RESEARCH LIBRARY</span>
            <h2>Notes from the lab.</h2>
          </div>
          <Link className="text-link" href="/research">
            All articles →
          </Link>
        </div>
        <div className="article-grid">
          {articles?.length ? (
            articles.slice(0, 3).map((a) => (
              <Link
                className="article-card"
                key={a.id}
                href={`/research/${a.slug}`}
              >
                {a.cover ? (
                  <img src={a.cover} alt="" />
                ) : (
                  <div className="article-art">Research notes</div>
                )}
                <span className="eyebrow">RESEARCH REFERENCE</span>
                <h3>{a.title}</h3>
                <p>{a.summary}</p>
                <span>Read article ↗</span>
              </Link>
            ))
          ) : (
            <div className="empty">
              {articles === null
                ? "The research library is temporarily unavailable."
                : "Research articles will appear here when published."}
            </div>
          )}
        </div>
      </section>
      <section className="section faq">
        <div>
          <span className="eyebrow">BEFORE YOU ENQUIRE</span>
          <h2>A few useful answers.</h2>
        </div>
        <div>
          {[
            [
              "Who are these materials for?",
              "Materials are intended strictly for laboratory research, not human or clinical use.",
            ],
            [
              "How do I request documentation?",
              "Contact our team with the product name and the documents you require.",
            ],
            [
              "How do I get a quotation?",
              "Send your specifications and required quantity directly to our team.",
            ],
          ].map(([q, a]) => (
            <details key={q}>
              <summary>{q}</summary>
              <p>{a}</p>
            </details>
          ))}
        </div>
      </section>
      <section className="contact-band">
        <h2>
          Let’s talk about
          <br />
          your next project.
        </h2>
        <ContactButton className="button">Start an enquiry ↗</ContactButton>
      </section>
    </>
  );
}
