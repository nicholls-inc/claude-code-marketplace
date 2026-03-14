/** Test: Pure functions → PURE_FUNCTION */

export function formatCurrency(amount: number, currency = "USD"): string {
  const symbols: Record<string, string> = { USD: "$", EUR: "€", GBP: "£" };
  return `${symbols[currency] || currency}${amount.toFixed(2)}`;
}

export function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\s-]/g, "")
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-")
    .trim();
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max);
}

export function groupBy<T>(items: T[], key: keyof T): Record<string, T[]> {
  const groups: Record<string, T[]> = {};
  for (const item of items) {
    const k = String(item[key]);
    (groups[k] = groups[k] || []).push(item);
  }
  return groups;
}

export function deepMerge(target: any, source: any): any {
  const result = { ...target };
  for (const key of Object.keys(source)) {
    if (source[key] && typeof source[key] === "object" && !Array.isArray(source[key])) {
      result[key] = deepMerge(result[key] || {}, source[key]);
    } else {
      result[key] = source[key];
    }
  }
  return result;
}

export function withLogging(value: number): number {
  console.log(`Value: ${value}`);
  return value * 2;
}
