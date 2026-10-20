export const consentKey = "shop-cookie-choice-v2";
export const consentEvent = "shop:cookie-choice";
export const consentLifetime = 180 * 24 * 60 * 60 * 1000;
export let consentStorageFailed = false;
export type Choice = "essential" | "accepted";
export function readConsent(scope: string) {
  if (consentStorageFailed) return null;
  try {
    const value = JSON.parse(localStorage.getItem(consentKey) || "null");
    if (value && value.scope === scope &&
      (value.choice === "essential" || value.choice === "accepted") &&
      Number.isFinite(value.savedAt) && value.savedAt <= Date.now() &&
      Date.now() - value.savedAt < consentLifetime) {
      return value as { choice: Choice; savedAt: number; scope: string };
    }
  } catch {}
  return null;
}
export function saveConsent(scope: string, choice: Choice) {
  try {
    localStorage.setItem(consentKey, JSON.stringify({ scope, choice, savedAt: Date.now() }));
    consentStorageFailed = false;
    window.dispatchEvent(new Event(consentEvent));
    return true;
  } catch {
    consentStorageFailed = true;
    try { localStorage.removeItem(consentKey); } catch {}
    window.dispatchEvent(new Event(consentEvent));
    return false;
  }
}
