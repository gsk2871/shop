"use client";
import { useState } from "react";
import Link from "next/link";
interface Banner {
	title: string;
	subtitle: string;
	image: string;
	link: string;
}
export default function Hero({ banners }: { banners: Banner[] }) {
	const [index, setIndex] = useState(0);
	const b = banners[index] || banners[0];
	if (!b) return null;
	return (
		<section className="hero">
			<div className="hero-copy">
				<span className="eyebrow">LABORATORY RESEARCH · B2B SUPPLY</span>
				<h1>{b.title}</h1>
				<p>{b.subtitle}</p>
				<div className="actions">
					<Link className="button" href={b.link || "/contact"}>
						Make an enquiry <span>↗</span>
					</Link>
					<Link className="text-link" href="/research">
						Explore research →
					</Link>
				</div>
				{banners.length > 1 && (
					<div className="dots" aria-label="Banner navigation">
						{banners.map((_, i) => (
							<button
								key={i}
								aria-label={`Show banner ${i + 1}`}
								aria-pressed={i === index}
								onClick={() => setIndex(i)}
							>
								{String(i + 1).padStart(2, "0")}
							</button>
						))}
					</div>
				)}
			</div>
			<div className="hero-art">
				{b.image ? (
					<img src={b.image} alt={b.title} />
				) : (
					<>
						<div className="orbit orbit-one" />
						<div className="orbit orbit-two" />
						<div className="vial">
							<div className="vial-cap" />
							<div className="vial-body">
								<span>
									RESEARCH
									<br />
									SUPPLY
								</span>
								<hr />
								<small>
									LABORATORY
									<br />
									USE ONLY
								</small>
							</div>
						</div>
						<span className="art-caption">A FOUNDATION FOR DISCOVERY</span>
					</>
				)}
			</div>
		</section>
	);
}
