"use client";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";

const storageKey = "shop-cookie-choice-v1";
const settingsEvent = "shop:cookie-settings";
const lifetime = 180 * 24 * 60 * 60 * 1000;
type Choice = "essential" | "accepted";

export function CookieSettingsButton() {
  return (
    <button
      type="button"
      className="footer-contact"
      onClick={() => window.dispatchEvent(new Event(settingsEvent))}
    >
      Cookie settings
    </button>
  );
}

export default function CookieNotice({
  title,
  content,
}: {
  title: string;
  content: string;
}) {
  const [show, setShow] = useState(false);
  const [managing, setManaging] = useState(false);
  const heading = useRef<HTMLHeadingElement>(null);
  const opener = useRef<HTMLElement | null>(null);
  useEffect(() => {
    try {
      const saved = JSON.parse(localStorage.getItem(storageKey) || "null");
      const valid =
        saved &&
        ["essential", "accepted"].includes(saved.choice) &&
        typeof saved.savedAt === "number" &&
        saved.savedAt <= Date.now() &&
        Date.now() - saved.savedAt < lifetime;
      setShow(!valid);
    } catch {
      setShow(true);
    }
    function openSettings() {
      opener.current =
        document.activeElement instanceof HTMLElement
          ? document.activeElement
          : null;
      setManaging(true);
      setShow(true);
    }
    window.addEventListener(settingsEvent, openSettings);
    return () => window.removeEventListener(settingsEvent, openSettings);
  }, []);
  useEffect(() => {
    if (show && managing) heading.current?.focus();
  }, [show, managing]);
  function choose(choice: Choice) {
    try {
      localStorage.setItem(
        storageKey,
        JSON.stringify({ choice, savedAt: Date.now() }),
      );
    } catch {}
    setShow(false);
    setManaging(false);
    opener.current?.focus();
  }
  if (!show) return null;
  return (
    <aside
      className="cookie cookie-notice"
      role="region"
      aria-labelledby="cookie-heading"
    >
      <button
        type="button"
        className="cookie-close"
        aria-label="Close and use necessary cookies only"
        onClick={() => choose("essential")}
      >
        ×
      </button>
      <div className="cookie-copy">
        <h2 id="cookie-heading" ref={heading} tabIndex={-1}>
          {managing
            ? "Manage cookies"
            : title.trim() || "We respect your privacy"}
        </h2>
        {managing ? (
          <>
            <p>
              <strong>Necessary storage — always active</strong>
            </p>
            <p>
              We store your cookie preference on this device. This website
              currently uses no optional analytics or advertising cookies.
            </p>
            <p>You can reopen these settings at any time from the footer.</p>
          </>
        ) : (
          <p>
            {content.trim() ||
              "We use essential storage to remember your preferences. Optional cookies are only used with your permission. You can review your choices at any time."}
          </p>
        )}
        <Link href="/privacy">More info</Link>
      </div>
      <div className="cookie-actions">
        <button
          type="button"
          className="button"
          onClick={() => choose("accepted")}
        >
          Accept All Cookies
        </button>
        <button
          type="button"
          className="outline"
          onClick={() => choose("essential")}
        >
          Necessary Cookies Only
        </button>
        {managing ? (
          <button
            type="button"
            className="cookie-manage"
            onClick={() => setManaging(false)}
          >
            Back
          </button>
        ) : (
          <button
            type="button"
            className="cookie-manage"
            onClick={() => setManaging(true)}
          >
            Manage Cookies
          </button>
        )}
      </div>
    </aside>
  );
}
