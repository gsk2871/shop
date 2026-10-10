"use client";
import { useState } from "react";
import Link from "next/link";
export default function Navigation() {
  const [open, setOpen] = useState(false);
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
          ["/products", "Catalogue"],
          ["/reports", "Documentation"],
          ["/research", "Research"],
          ["/about", "About"],
        ].map(([path, name]) => (
          <Link href={path} key={path} onClick={() => setOpen(false)}>
            {name}
          </Link>
        ))}
      </nav>
    </>
  );
}
