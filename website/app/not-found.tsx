import Link from "next/link";
export default function NotFound() {
	return (
		<section className="section prose">
			<h1>Page not found.</h1>
			<p>The content may be unavailable or unpublished.</p>
			<Link className="button" href="/">
				Return home →
			</Link>
		</section>
	);
}
