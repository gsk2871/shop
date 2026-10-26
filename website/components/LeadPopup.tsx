"use client";
import { useEffect, useRef, useState, type FormEvent, type MouseEvent } from "react";
import type { PublicLeadPopup } from "../lib/contracts";

const sessionKey = "shop.lead-popup.seen";
function isBackdrop(event: MouseEvent<HTMLDialogElement>) {
  if (event.target !== event.currentTarget) return false;
  const bounds = event.currentTarget.getBoundingClientRect();
  return event.clientX < bounds.left || event.clientX > bounds.right ||
    event.clientY < bounds.top || event.clientY > bounds.bottom;
}

export default function LeadPopup({ config }: { config: PublicLeadPopup | null }) {
  const dialog = useRef<HTMLDialogElement>(null);
  const opener = useRef<HTMLButtonElement>(null);
  const backdropPressed = useRef(false);
  const checked = useRef(false);
  const pending = useRef(false);
  const [open, setOpen] = useState(false);
  const [contact, setContact] = useState("");
  const [busy, setBusy] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => {
    if (!config?.enabled || checked.current) return;
    checked.current = true;
    try {
      if (sessionStorage.getItem(sessionKey)) return;
      sessionStorage.setItem(sessionKey, "1");
    } catch { /* Storage-disabled visitors still get one popup per document. */ }
    setOpen(true);
  }, [config?.enabled]);
  useEffect(() => {
    const node = dialog.current;
    if (!node || !open) return;
    const previous = document.activeElement;
    const overflow = document.body.style.overflow;
    node.showModal();
    node.querySelector<HTMLElement>("#lead-popup-title")?.focus();
    document.body.style.overflow = "hidden";
    return () => {
      node.close();
      document.body.style.overflow = overflow;
      if (previous instanceof HTMLElement && previous.isConnected) previous.focus();
      else opener.current?.focus();
    };
  }, [open]);
  async function submit(e: FormEvent) {
    e.preventDefault();
    if (pending.current) return;
    const value = contact.trim();
    const valid = value.includes("@")
      ? /^[^\s@<>]+@[^\s@<>]+\.[^\s@<>]+$/.test(value)
      : /^\+[1-9][0-9]{6,14}$/.test(value.replace(/[ ()-]/g, ""));
    if (!valid) { setError("Enter a valid email or WhatsApp number including the country code (for example, +1 202 555 0123)."); return; }
    pending.current = true; setBusy(true); setError("");
    try {
      const response = await fetch("/api/leads", {
        method: "POST", headers: { "Content-Type": "application/json" },
        credentials: "omit", body: JSON.stringify({ contact: value }), signal: AbortSignal.timeout(15000),
      });
      if (!response.ok) {
        if (response.status === 429) throw new Error("Too many attempts. Please try again in 10 minutes.");
        if (response.status === 400) throw new Error("Please check your email or WhatsApp number, including the country code.");
        if (response.status === 409) throw new Error("This offer is currently unavailable. Please contact our team.");
        throw new Error("We couldn't save your details. Please try again.");
      }
      setSuccess(true); setContact("");
    } catch (e) {
      setError(e instanceof Error && e.name !== "TimeoutError" && e.name !== "TypeError" ? e.message : "We couldn't save your details. Please try again.");
    } finally { pending.current = false; setBusy(false); }
  }
  if (!config?.enabled) return null;
  return (
    <>
    {!open && <button type="button" ref={opener} className="lead-popup-reopen" aria-haspopup="dialog" aria-controls="lead-popup" onClick={() => setOpen(true)}>Special offer</button>}
    <dialog id="lead-popup" ref={dialog} className="lead-popup" aria-labelledby="lead-popup-title" aria-describedby="lead-popup-description"
      onCancel={(e) => { e.preventDefault(); setOpen(false); }}
      onPointerDown={(e) => { backdropPressed.current = e.isPrimary && e.button === 0 && isBackdrop(e); }}
      onPointerCancel={() => { backdropPressed.current = false; }}
      onClick={(e) => {
        // Both press and release must be outside; dragging from the form must not dismiss it.
        const dismiss = backdropPressed.current && isBackdrop(e);
        backdropPressed.current = false;
        if (dismiss) setOpen(false);
      }}>
      <button type="button" className="lead-popup-close" aria-label="Close" onClick={() => setOpen(false)}>×</button>
      <div aria-live="polite">
        <h2 id="lead-popup-title" tabIndex={-1}>{success ? config.successTitle : config.title}</h2>
        <p id="lead-popup-description" className="lead-popup-description">{success ? config.successDescription : config.description}</p>
      </div>
      {!success && <form onSubmit={submit}>
        <label className="visually-hidden lead-contact-label" htmlFor="lead-contact">{config.placeholder}</label>
        <input id="lead-contact" type="text" maxLength={254} required value={contact} placeholder={config.placeholder} onChange={(e) => setContact(e.target.value)} aria-invalid={!!error} aria-describedby={error ? "lead-error" : undefined} autoCapitalize="none" spellCheck={false} />
        {error && <p id="lead-error" className="lead-popup-error" role="alert">{error}</p>}
        <button className="lead-popup-submit" type="submit" disabled={busy} aria-busy={busy}>{busy ? "Submitting…" : config.submitText}</button>
      </form>}
      {!success && <>
        <button type="button" className="lead-popup-dismiss" onClick={() => setOpen(false)}>{config.dismissText}</button>
        <p className="lead-popup-notice">{config.notice}</p>
      </>}
    </dialog>
    </>
  );
}
