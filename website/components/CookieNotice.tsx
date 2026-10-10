"use client";
import { useEffect, useState } from "react";
export default function CookieNotice({ title, content }: { title: string; content: string }) {
	const [show, setShow] = useState(false);
	useEffect(() => {
		try {
			setShow(!localStorage.getItem("cookie-choice"));
		} catch {
			setShow(true);
		}
	}, []);
	function choose(value: string) {
		try {
			localStorage.setItem("cookie-choice", value);
		} catch {}
		setShow(false);
	}
	return show && (title || content) ? (
		<aside className="cookie" aria-label="Cookie preferences">
			<div>
				<strong>{title}</strong>
				<p>{content}</p>
			</div>
			<button className="outline" onClick={() => choose("essential")}>
				Essential only
			</button>
			<button className="button" onClick={() => choose("accepted")}>
				Accept
			</button>
		</aside>
	) : null;
}
