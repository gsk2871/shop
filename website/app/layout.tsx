import "@shop/content/styles.css";
import type { Metadata } from "next";
import type { CSSProperties, ReactNode } from "react";
import Link from "next/link";
import { settings } from "../lib/content";
import ContactButton from "../components/ContactButton";
import Navigation from "../components/Navigation";
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
    "--primary": s.theme.primary,
    "--surface": s.theme.surface,
    "--text": s.theme.text,
    "--muted": s.theme.muted,
    "--border": s.theme.border,
    "--accent": s.theme.accent,
    "--on-primary": s.theme.onPrimary,
    "--canvas": s.theme.background,
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
          <Navigation />
          <ContactButton className="nav-contact">Contact us ↗</ContactButton>
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
            <ContactButton className="footer-contact">
              Contact on WhatsApp ↗
            </ContactButton>
            <a href="https://db-ip.com" target="_blank" rel="noreferrer">
              IP Geolocation by DB-IP
            </a>
            <Link href="/privacy">Privacy & cookies</Link>
          </div>
          <p className="footer-note">{s.disclaimer}</p>
        </footer>
        <ContactButton className="whatsapp">WhatsApp ↗</ContactButton>
        <CookieNotice />
      </body>
    </html>
  );
}
