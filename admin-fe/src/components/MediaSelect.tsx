import { useEffect, useState } from "react";
import { Select } from "antd";
import { api, type Media } from "../services/cms";
export default function MediaSelect({
	value,
	onChange,
}: {
	value?: string;
	onChange?: (v: string) => void;
}) {
	const [rows, setRows] = useState<Media[]>([]);
	const [error, setError] = useState("");
	useEffect(() => {
		api<Media[]>("media")
			.then(setRows)
			.catch((e: Error) => setError(e.message));
	}, []);
	return (
		<Select
			allowClear
			showSearch
			optionFilterProp="label"
			value={value || undefined}
			onChange={(v) => onChange?.(v ?? "")}
			placeholder={error || "从素材库选择图片"}
			options={rows
				.filter((r) => r.mime.startsWith("image/"))
				.map((r) => ({ label: r.name, value: r.url }))}
		/>
	);
}
