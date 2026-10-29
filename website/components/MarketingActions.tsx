import Link from "next/link";
import ContactButton from "./ContactButton";

export default function MarketingActions() {
  return <div className="marketing-actions">
    <ContactButton className="button marketing-primary">💬 Get Pricing on WhatsApp</ContactButton>
    <Link className="button marketing-products" href="/products">🧪 View Products</Link>
  </div>;
}
