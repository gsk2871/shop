"use client";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";

import { consentEvent, consentKey, readConsent, saveConsent, type Choice } from "../lib/ads-consent";
import { useGoogleAds } from "./GoogleAds";
const settingsEvent = "shop:cookie-settings";


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
  const { scope, enabled } = useGoogleAds();
  const [saveError, setSaveError] = useState(false);
  const [show, setShow] = useState(false);
  const [managing, setManaging] = useState(false);
  const heading = useRef<HTMLHeadingElement>(null);
  const opener = useRef<HTMLElement | null>(null);
  useEffect(() => {
    function refresh() { setShow(!readConsent(scope)); }
    function storage(event: StorageEvent) { if (event.key === consentKey || event.key === null) refresh(); }
    refresh();
    window.addEventListener(consentEvent, refresh);
    window.addEventListener("storage", storage);
    window.addEventListener("focus", refresh);
    function openSettings() {
      opener.current =
        document.activeElement instanceof HTMLElement
          ? document.activeElement
          : null;
      setManaging(true);
      setShow(true);
    }
    window.addEventListener(settingsEvent, openSettings);
    return () => {
      window.removeEventListener(settingsEvent, openSettings);
      window.removeEventListener(consentEvent, refresh);
      window.removeEventListener("storage", storage);
      window.removeEventListener("focus", refresh);
    };
  }, [scope]);
  useEffect(() => {
    if (show && managing) heading.current?.focus();
  }, [show, managing]);
  function choose(choice: Choice) {
    if (!saveConsent(scope, choice)) {
      setSaveError(true);
      return;
    }
    setSaveError(false);
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
              We store your cookie preference on this device. Your choice is stored on this device for up to 180 days.
            </p>
            <p>You can reopen these settings at any time from the footer.</p>
          </>
        ) : (
          <p>
            {content.trim() ||
              "We use essential storage to remember your preferences. Optional cookies are only used with your permission. You can review your choices at any time."}
          </p>
        )}
        {enabled && <p>With your permission, we use Google Ads cookies and share measurement data with Google to measure whether ads lead to WhatsApp clicks. We do not enable personalized advertising. Rejecting does not affect browsing or contacting us. You can withdraw consent at any time using Cookie settings in the footer.</p>}
        {saveError && <p role="alert">Your preference could not be saved. Further conversion reporting is disabled in this tab. Please allow site storage and try again.</p>}
        <Link href="/privacy">More info</Link>
        {enabled && <> · <a href="https://business.safety.google/privacy/" target="_blank" rel="noreferrer">How Google uses data</a></>}
      </div>
      <div className="cookie-actions">
        <button
          type="button"
          className="button"
          onClick={() => choose("accepted")}
        >
          {enabled ? "Accept advertising measurement" : "Accept All Cookies"}
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
