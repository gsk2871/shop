import Link from "next/link";
export const metadata = { title: "Catalogue" };
export default function Products() {
	return (
		<section className="section prose">
			<span className="eyebrow">RESEARCH CATALOGUE</span>
			<h1>Find your starting point.</h1>
			<p>
				Our online catalogue is being prepared. Contact the team for available
				products and specifications.
			</p>
			<Link href="/contact" className="button">
				Request product information ↗
			</Link>
		</section>
	);
}
