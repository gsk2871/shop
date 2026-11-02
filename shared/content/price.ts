/** Change only the USD label; preserve the backend's exact amount formatting. */
export function formatDisplayPrice(value: string): string {
  return value.replace(/^USD\s+/, "$");
}
