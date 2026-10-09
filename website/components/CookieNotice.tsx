"use client";
import { useEffect, useState } from "react";
export default function CookieNotice() {
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
	return show ? (
		<aside className="cookie" aria-label="Cookie preferences">
			<div>
				<strong>Your privacy matters.</strong>
				<p>
					We use essential storage for your preferences. Optional analytics are
					currently disabled.
				</p>
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
