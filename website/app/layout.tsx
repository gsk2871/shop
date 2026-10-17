import "@shop/content/styles.css";
import type { Metadata } from "next";
import type { CSSProperties, ReactNode } from "react";
import Link from "next/link";
import { settings } from "../lib/content";
import ContactButton from "../components/ContactButton";
import SectionRuler from "../components/SectionRuler";
import Navigation from "../components/Navigation";
import SocialLinks from "../components/SocialLinks";
import CookieNotice, { CookieSettingsButton } from "../components/CookieNotice";
import "./globals.css";
export const dynamic = "force-dynamic";
export async function generateMetadata(): Promise<Metadata> {
  const s = await settings();
  return {
    title: s.brand ? { default: s.brand, template: `%s | ${s.brand}` } : "Home",
    description: s.description,
    icons: s.logo ? { icon: s.logo } : undefined,
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
        <header className="header">
          <Link className="brand" href="/" aria-label={s.brand || "Home"}>
            {s.logo ? (
              <img src={s.logo} alt={s.brand} />
            ) : s.brandMark ? (
              <span className="brand-symbol">{s.brandMark}</span>
            ) : null}
            <span>{s.brand}</span>
          </Link>
          <Navigation />
          <ContactButton className="nav-contact">Contact us ↗</ContactButton>
        </header>
        <main>{children}</main>
        <SectionRuler />
        <footer className="site-footer">
          {(s.brand || s.description) && (
            <div className="footer-identity">
              {s.brand && (
                <Link
                  className="footer-brand"
                  href="/"
                  aria-label={s.brand || "Home"}
                >
                  {s.brand}
                </Link>
              )}
              {s.description && <p>{s.description}</p>}
            </div>
          )}
          <nav className="footer-group" aria-label="Footer navigation">
            <span className="eyebrow">EXPLORE</span>
            <Link href="/">Home</Link>
            <Link href="/products">Products</Link>
            <Link href="/news">News</Link>
            <Link href="/faq">FAQ</Link>
          </nav>
          <div className="footer-group">
            <span className="eyebrow">GET IN TOUCH</span>
            <ContactButton className="footer-contact">
              Contact on WhatsApp ↗
            </ContactButton>
            <Link href="/privacy">Privacy & cookies</Link>
            <CookieSettingsButton />
            <SocialLinks accounts={s.socialAccounts || []} />
          </div>
        </footer>
        <ContactButton className="whatsapp">WhatsApp ↗</ContactButton>
        <CookieNotice title={s.cookieTitle} content={s.cookieContent} />
      </body>
    </html>
  );
}
