import { settings } from "../../lib/content";
export async function generateMetadata() {
  const s = await settings();
  return { title: s.privacyTitle || "Privacy" };
}
export default async function Privacy() {
  const s = await settings();
  return (
    <section className="section prose">
      {s.privacyTitle && <h1>{s.privacyTitle}</h1>}
      <div className="configured-copy">{s.privacyContent}</div>
    </section>
  );
}
