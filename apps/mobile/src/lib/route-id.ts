/**
 * parseId reads a scan's or a recipe's id from its route: a whole number above zero, or undefined
 * for anything else, so a screen never asks the api about one that can't exist.
 */
export function parseId(param: string | undefined): bigint | undefined {
  if (param === undefined || !/^\d+$/.test(param)) {
    return undefined;
  }
  const id = BigInt(param);
  return id > 0n ? id : undefined;
}
