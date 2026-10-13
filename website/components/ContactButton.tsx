"use client";
import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import type { WhatsAppContact } from "../lib/contracts";
// Image addresses are resolved by the trusted backend from a media ID.
function validImageURL(raw: string) {
  if (/^\/uploads\/[a-f0-9]+\/[a-f0-9]+\.(png|jpg|webp|gif)$/.test(raw)) return true;
  try { const u = new URL(raw); return u.protocol === 'https:' && !u.username && !u.password && !u.search && !u.hash; } catch { return false; }
}
export default function ContactButton({
  children = "Contact on WhatsApp ↗",
  className = "button",
  productName,
}: {
  children?: ReactNode;
  className?: string;
  productName?: string;
}) {
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [contact, setContact] = useState<WhatsAppContact>();
  const [error, setError] = useState(false);
  const [imageError, setImageError] = useState(false);
  const dialog = useRef<HTMLDialogElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const pending = useRef<AbortController | null>(null);
  const title = useId();
  function close() {
    pending.current?.abort();
    pending.current = null;
    setOpen(false);
    setBusy(false);
  }
  useEffect(() => {
    if (!open) return;
    const el = dialog.current;
    const scroll = window.scrollY;
    const previous = document.body.style.overflow;
    el?.showModal();
    document.body.style.overflow = "hidden";
    return () => {
      pending.current?.abort();
      el?.close();
      document.body.style.overflow = previous;
      window.scrollTo(0, scroll);
      trigger.current?.focus();
    };
  }, [open]);
  async function load() {
    if (pending.current) return;
    const controller = new AbortController();
    pending.current = controller;
    setBusy(true);
    setError(false);
    setContact(undefined);
    setImageError(false);
    const timeout = setTimeout(() => controller.abort(), 15000);
    try {
      const response = await fetch("/api/contact", {
        method: "GET",
        cache: "no-store",
        credentials: "same-origin",
        signal: controller.signal,
      });
      if (!response.ok) throw new Error("Unavailable");
      const value: WhatsAppContact = await response.json();
      const url = new URL(value.url);
      if (
        !/^[1-9][0-9]{6,14}$/.test(value.phoneNumber) ||
        url.protocol !== "https:" ||
        url.hostname !== "wa.me" ||
        url.port ||
        url.username ||
        url.password ||
        url.pathname !== `/${value.phoneNumber}` ||
        url.search ||
        url.hash ||
        !validImageURL(value.qrCodeUrl)
      )
        throw new Error("Invalid contact");
      if (productName)
        url.searchParams.set(
          "text",
          `Hello, I would like to enquire about ${productName}.`,
        );
      if (pending.current === controller)
        setContact({ ...value, url: url.toString() });
    } catch {
      if (pending.current === controller) setError(true);
    } finally {
      clearTimeout(timeout);
      if (pending.current === controller) {
        pending.current = null;
        setBusy(false);
      }
    }
  }
  return (
    <>
      <button
        ref={trigger}
        type="button"
        className={className}
        onClick={() => {
          setOpen(true);
          void load();
        }}
      >
        {children}
      </button>
      <dialog
        ref={dialog}
        className="contact-dialog"
        aria-labelledby={title}
        onCancel={(e) => {
          e.preventDefault();
          close();
        }}
        onClick={(e) => {
          if (e.target === dialog.current) {
            const rect = dialog.current.getBoundingClientRect();
            if (
              e.clientX < rect.left ||
              e.clientX > rect.right ||
              e.clientY < rect.top ||
              e.clientY > rect.bottom
            )
              close();
          }
        }}
      >
        <div className="contact-dialog-heading">
          <h2 id={title}>Contact on WhatsApp</h2>
          <button
            type="button"
            className="contact-close"
            aria-label="Close contact window"
            onClick={close}
          >
            ×
          </button>
        </div>
        <div aria-live="polite" aria-busy={busy}>
          {busy && <p>Finding a contact for you…</p>}
          {error && (
            <>
              <p role="alert">
                WhatsApp contact is temporarily unavailable. Please try again
                later.
              </p>
              <button
                type="button"
                className="button"
                onClick={() => void load()}
                disabled={busy}
              >
                Try again
              </button>
            </>
          )}
          {contact && (
            <>
              <p className="contact-account">
                {contact.accountName}
                <br />
                <span>+{contact.phoneNumber}</span>
              </p>
              <a
                className="button contact-chat"
                href={contact.url}
                target="_blank"
                rel="noreferrer"
              >
                Open WhatsApp ↗
              </a>
              {imageError ? (
                <p>
                  QR image could not be loaded. You can still open WhatsApp
                  above.
                </p>
              ) : (
                <img
                  className="contact-qr"
                  src={contact.qrCodeUrl}
                  alt={`WhatsApp QR code for ${contact.accountName}`}
                  onError={() => setImageError(true)}
                />
              )}
              <p className="subtle">
                Open WhatsApp directly, or scan the QR code on another device.
              </p>
            </>
          )}
        </div>
      </dialog>
    </>
  );
}
