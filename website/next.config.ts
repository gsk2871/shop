import type { NextConfig } from "next";
const backend = process.env.BACKEND_URL || "http://127.0.0.1:8080";
const config: NextConfig = {
	output: "standalone",
	async rewrites() {
		return [
			{ source: "/uploads/:path*", destination: `${backend}/uploads/:path*` },
		];
	},
};
export default config;
