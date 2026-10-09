import { settings } from "../../lib/content";
export const metadata = { title: "Contact" };
export default async function Contact() {
	const s = await settings();
	return (
		<section className="section prose">
			<span className="eyebrow">LET’S START A CONVERSATION</span>
			<h1>
				Your research.
				<br />
				Our attention.
			</h1>
			<p>
				Share the product name, specifications and required quantity. Our team
				will help you confirm availability and documentation.
			</p>
			<div className="contact-options">
				{s.email && (
					<a className="button" href={`mailto:${s.email}`}>
						Email {s.email} ↗
					</a>
				)}
				{s.whatsapp && (
					<a
						className="outline"
						href={`https://wa.me/${s.whatsapp}`}
						target="_blank"
						rel="noreferrer"
					>
						Contact on WhatsApp ↗
					</a>
				)}
				{!s.email && !s.whatsapp && (
					<p>Contact details are being updated. Please check back shortly.</p>
				)}
			</div>
			<p className="subtle">
				All enquiries must relate to laboratory research use.
			</p>
		</section>
	);
}
