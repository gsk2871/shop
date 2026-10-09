import { useEffect, useState, type ReactNode } from "react";
import { history, useLocation } from "@umijs/max";
import { Alert, Spin } from "antd";
import { api } from "../services/cms";
export default function Auth({ children }: { children: ReactNode }) {
	const [ready, setReady] = useState(false);
	const [error, setError] = useState("");
	const location = useLocation();
	useEffect(() => {
		let active = true;
		api("me")
			.then(() => {
				if (active) setReady(true);
			})
			.catch((e: Error) => {
				if (!active) return;
				setError(e.message);
				history.replace("/login");
			});
		return () => {
			active = false;
		};
	}, [location.pathname]);
	if (error) return <Alert type="error" message={error} />;
	if (!ready) return <Spin />;
	return <>{children}</>;
}
