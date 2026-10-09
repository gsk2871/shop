import type { Metadata } from "next";
import type { CSSProperties, ReactNode } from "react";
import Link from "next/link";
import { settings } from "../lib/content";
import CookieNotice from "../components/CookieNotice";
import "./globals.css";
export const dynamic = "force-dynamic";
export async function generateMetadata(): Promise<Metadata> {
	const s = await settings();
	return {
		title: { default: s.brand, template: `%s | ${s.brand}` },
		description: s.description,
	};
}
export default async function Layout({ children }: { children: ReactNode }) {
	const s = await settings();
	const style = {
		"--primary": s.primaryColor,
		"--canvas": s.backgroundColor,
	} as CSSProperties;
	return (
		<html lang="en">
			<body style={style}>
				<div className="notice">{s.disclaimer}</div>
				<header className="header">
					<Link className="brand" href="/">
						{s.logo ? (
							<img src={s.logo} alt={s.brand} />
						) : (
							<span className="brand-symbol">
								R<span>·</span>
							</span>
						)}
						<span>{s.brand}</span>
					</Link>
					<nav aria-label="Main navigation">
						<Link href="/products">Catalogue</Link>
						<Link href="/reports">Documentation</Link>
						<Link href="/research">Research</Link>
						<Link href="/about">About</Link>
					</nav>
					<Link className="nav-contact" href="/contact">
						Contact us ↗
					</Link>
				</header>
				<main>{children}</main>
				<footer>
					<div>
						<Link className="footer-brand" href="/">
							{s.brand}
						</Link>
						<p>{s.description}</p>
					</div>
					<div>
						<span className="eyebrow">EXPLORE</span>
						<Link href="/products">Catalogue</Link>
						<Link href="/research">Research library</Link>
						<Link href="/reports">Documentation</Link>
					</div>
					<div>
						<span className="eyebrow">GET IN TOUCH</span>
						{s.email && <a href={`mailto:${s.email}`}>{s.email}</a>}
						<Link href="/contact">Contact information →</Link>
						<Link href="/privacy">Privacy & cookies</Link>
					</div>
					<p className="footer-note">{s.disclaimer}</p>
				</footer>
				{s.whatsapp && (
					<a
						className="whatsapp"
						href={`https://wa.me/${s.whatsapp}`}
						target="_blank"
						rel="noreferrer"
						aria-label="Contact us on WhatsApp"
					>
						WhatsApp ↗
					</a>
				)}
				<CookieNotice />
			</body>
		</html>
	);
}
