"use client";
import { createContext, useContext, useEffect, useRef, type ReactNode } from "react";
import type { GoogleAdsSettings } from "../lib/contracts";
import { consentEvent, consentKey, consentLifetime, consentStorageFailed, readConsent } from "../lib/ads-consent";

type Tag = (...args: unknown[]) => void;
type AdsWindow = Window & { dataLayer?: unknown[]; gtag?: Tag };
const AdsContext = createContext({ scope: "", enabled: false, track: () => {} });
export const useGoogleAds = () => useContext(AdsContext);
export function validAds(config: GoogleAdsSettings) {
  return config.enabled === true && /^AW-[0-9]{1,20}$/.test(config.id) &&
    /^[A-Za-z0-9_-]{1,100}$/.test(config.conversionLabel);
}
function clearAdsCookies() {
  // Only remove Google Ads first-party cookies, never unrelated site cookies.
  const labels = location.hostname.split(".");
  for (const cookie of document.cookie.split(";")) {
    const name = cookie.split("=")[0].trim();
    if (!/^_gcl_[A-Za-z0-9_]+$/.test(name)) continue;
    document.cookie = `${name}=; Max-Age=0; path=/`;
    for (let i = 0; i < labels.length - 1; i++) {
      document.cookie = `${name}=; Max-Age=0; path=/; domain=${labels.slice(i).join(".")}`;
    }
  }
}
export default function GoogleAds({ config, noticeVersion, children }: {
  config: GoogleAdsSettings; noticeVersion: string; children: ReactNode;
}) {
  const enabled = validAds(config);
  // Bind consent to the recipient, purpose and displayed configurable notice.
  const scope = JSON.stringify(["ads-measurement-v1", enabled, config.id, config.conversionLabel, noticeVersion]);
  const track = useRef<() => void>(() => {});
  useEffect(() => {
    const w = window as AdsWindow;
    let script: HTMLScriptElement | undefined;
    let ready = false;
    let pending = 0;
    let failed = false;
    let disposed = false;
    let expiry: ReturnType<typeof setTimeout> | undefined;
    const permitted = () => enabled && readConsent(scope)?.choice === "accepted";
    function send() {
      if (permitted() && ready) w.gtag?.("event", "conversion", {
        send_to: `${config.id}/${config.conversionLabel}`,
      });
    }
    function sync() {
      clearTimeout(expiry);
      if (!permitted()) {
        pending = 0;
        script?.remove();
        script = undefined;
        if (ready) {
          // A loaded third-party runtime cannot be unloaded. Reload into the
          // denied state after clearing its queue and our first-party cookies.
          ready = false;
          w.gtag?.("consent", "update", { ad_storage: "denied", ad_user_data: "denied", ad_personalization: "denied", analytics_storage: "denied" });
          w.dataLayer?.splice(0);
          w.gtag = () => {};
          clearAdsCookies();
          if (!consentStorageFailed) window.location.reload();
        }
        return;
      }
      const saved = readConsent(scope)!;
      expiry = setTimeout(sync, Math.min(2147483647, Math.max(1, saved.savedAt + consentLifetime - Date.now())));
      if (script || ready || failed) return;
      const current = document.createElement("script");
      script = current;
      current.async = true;
      current.src = `https://www.googletagmanager.com/gtag/js?id=${config.id}`;
      // Queue consent defaults before the downloaded tag can execute.
      w.dataLayer = [];
      w.gtag = function () { w.dataLayer!.push(arguments); };
      w.gtag("consent", "default", {
        ad_storage: "denied", ad_user_data: "denied",
        ad_personalization: "denied", analytics_storage: "denied",
      });
      current.onload = () => {
        if (disposed || current !== script || !permitted()) return;
        w.gtag!("consent", "update", { ad_storage: "granted", ad_user_data: "granted" });
        w.gtag!("js", new Date());
        w.gtag!("config", config.id, { send_page_view: false, allow_ad_personalization_signals: false });
        ready = true;
        const count = pending;
        pending = 0;
        for (let i = 0; i < count; i++) send();
      };
      current.onerror = () => { failed = true; pending = 0; };
      document.head.appendChild(current);
    }
    track.current = () => {
      try {
        if (!permitted()) { sync(); return; }
        sync();
        if (ready) send();
        else if (!failed && pending < 20) pending++;
      } catch { /* Tracking must never prevent opening WhatsApp. */ }
    };
    const storage = (event: StorageEvent) => {
      if (event.key === consentKey || event.key === null) sync();
    };
    window.addEventListener(consentEvent, sync);
    window.addEventListener("storage", storage);
    window.addEventListener("focus", sync);
    sync();
    return () => {
      disposed = true;
      track.current = () => {};
      clearTimeout(expiry);
      window.removeEventListener(consentEvent, sync);
      window.removeEventListener("storage", storage);
      window.removeEventListener("focus", sync);
      script?.remove();
      if (ready) { clearAdsCookies(); window.location.reload(); }
    };
  }, [enabled, scope, config.id, config.conversionLabel]);
  return <AdsContext.Provider value={{ scope, enabled, track: () => track.current() }}>{children}</AdsContext.Provider>;
}
