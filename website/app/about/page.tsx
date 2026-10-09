import { settings } from "../../lib/content";
import Link from "next/link";
export const metadata = { title: "About" };
export default async function About() {
	const s = await settings();
	return (
		<section className="section prose">
			<span className="eyebrow">ABOUT {s.brand.toUpperCase()}</span>
			<h1>
				Built around
				<br />
				your research.
			</h1>
			<p className="intro">{s.description}</p>
			<p>
				Contact us to discuss product information, available documentation and
				your laboratory material requirements.
			</p>
			<Link href="/contact" className="button">
				Talk to our team ↗
			</Link>
		</section>
	);
}
