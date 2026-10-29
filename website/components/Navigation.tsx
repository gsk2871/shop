"use client";
import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
export default function Navigation() {
  const [open, setOpen] = useState(false);
  const pathname = usePathname();
  return (
    <>
      <button
        type="button"
        className="mobile-menu-toggle"
        aria-expanded={open}
        aria-controls="main-navigation"
        onClick={() => setOpen(!open)}
      >
        {open ? "Close menu" : "Menu"}
      </button>
      <nav
        id="main-navigation"
        className={open ? "mobile-open" : ""}
        aria-label="Main navigation"
      >
        {[
          ["/", "Home"],
          ["/products", "Products"],
          ["/news", "News"],
          ["/faq", "FAQ"],
          ["/about", "About"],
        ].map(([path, name]) => (
          <Link aria-current={pathname === path || (path !== "/" && pathname.startsWith(path + "/")) ? "page" : undefined} href={path} key={path} onClick={() => setOpen(false)}>
            {name}
          </Link>
        ))}
      </nav>
    </>
  );
}
