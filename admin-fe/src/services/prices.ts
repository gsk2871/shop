// Validate decimal input in integer minor units, matching the server's BIGINT prices.
export function validatePrices(original: string | null | undefined, sale: string | null | undefined, minorUnit: number) {
  if (!original && !sale) return;
  if (!original || !sale) throw new Error("原价和优惠价必须同时填写");
  function amount(raw: string) {
    if (!/^\d+(?:\.\d+)?$/.test(raw) || raw.length > 100) throw new Error("请输入有效的非负价格");
    const [whole, fraction = ""] = raw.split(".");
    if (fraction.length > minorUnit) throw new Error(`价格最多保留 ${minorUnit} 位小数`);
    const value = BigInt(whole + fraction.padEnd(minorUnit, "0"));
    if (value > 9223372036854775807n) throw new Error("价格超出支持范围");
    return value;
  }
  if (amount(sale) > amount(original)) throw new Error("优惠价必须小于或等于原价");
}
