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
      {s.googleAds.enabled && <>
        <h2>Advertising measurement and cookie choices</h2>
        <p>With your consent, we use Google Ads to measure whether our advertisements lead to clicks on Open WhatsApp. Google receives measurement information, which may include page and ad interaction information, online identifiers and browser/device information. A click does not confirm that you sent a message or made a purchase. We do not send WhatsApp message content or enable personalized advertising through this feature.</p>
        <p>Google advertising tags are blocked until you accept advertising measurement. Rejecting does not affect browsing or contacting us. Your choice is stored only in this browser for up to 180 days; clearing site storage removes it. You can change or withdraw your choice at any time using Cookie settings in the footer. Withdrawal stops future tracking; it does not undo data already sent.</p>
        <p>Google Ads may use first-party cookies such as _gcl_aw to associate an ad interaction with a conversion. Their lifetime is typically 90 days and may depend on Google configuration. Our cookie preference storage is separate from these advertising cookies.</p>
        <p><a href="https://business.safety.google/privacy/" target="_blank" rel="noreferrer">How Google uses your data</a> · <a href="https://policies.google.com/technologies/cookies" target="_blank" rel="noreferrer">Google cookie information</a></p>
      </> }
    </section>
  );
}
