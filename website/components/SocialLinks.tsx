"use client";

import { usePathname } from "next/navigation";
import type { PublicSocialAccount } from "../lib/contracts";

export default function SocialLinks({ accounts }: { accounts: PublicSocialAccount[] }) {
  const pathname = usePathname();
  if (pathname !== "/" || !accounts.length) return null;
  return (
    <nav className="footer-social-links" aria-label="Social media">
      {accounts.map((account, index) => (
        <a key={`${account.platform}-${index}`} href={account.link}
          target="_blank" rel="noopener noreferrer"
          aria-label={`${account.name} (${account.platform})`} title={account.name}>
          <img src={account.icon} alt="" width={24} height={24} loading="lazy" />
        </a>
      ))}
    </nav>
  );
}
