export async function getReadiness(): Promise<{ status: string }> {
	const response = await fetch("/backend/readyz", {
		signal: AbortSignal.timeout(5000),
	});
	if (!response.ok) throw new Error("Backend unavailable");
	return response.json();
}
