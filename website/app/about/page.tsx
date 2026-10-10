import { settings } from "../../lib/content";
export async function generateMetadata() {
  const s = await settings();
  return { title: s.aboutTitle || "About" };
}
export default async function About() {
  const s = await settings();
  return (
    <section className="section prose">
      {s.aboutTitle && <h1>{s.aboutTitle}</h1>}
      <div className="configured-copy">{s.aboutContent}</div>
    </section>
  );
}
