/** groupDigits writes a count with thousands separators, as the screens do ("1,240"). */
export function groupDigits(value: bigint) {
  return String(value).replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}
