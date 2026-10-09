import Link from "next/link";
export const metadata = { title: "Documentation" };
export default function Reports() {
	return (
		<section className="section prose">
			<span className="eyebrow">PRODUCT DOCUMENTATION</span>
			<h1>Details you can review.</h1>
			<p>
				Public reports will be listed here when available. Please contact the
				team to request documentation for a specific product.
			</p>
			<Link href="/contact" className="button">
				Request documentation ↗
			</Link>
		</section>
	);
}
