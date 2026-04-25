export function compactNumber(value: number) {
  return new Intl.NumberFormat("en", {
    notation: "compact",
    maximumFractionDigits: 1
  }).format(value);
}

export function percent(value: number, total: number) {
  if (total === 0) {
    return 0;
  }

  return Math.min(100, Math.round((value / total) * 100));
}
