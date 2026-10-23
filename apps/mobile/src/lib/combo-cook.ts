import type { IngredientOwner } from "@fries-in-the-bag/api";

/** bringItems packs the guest's required food that is absent from the chosen kitchen. */
export function bringItems(
  owners: readonly Pick<IngredientOwner, "accountId" | "ingredient">[],
  viewer: bigint | undefined,
  kitchen: bigint | undefined,
): string[] {
  if (viewer === undefined || kitchen === undefined || viewer === kitchen) return [];
  const available = new Set(
    owners.filter((item) => item.accountId === kitchen).map((item) => item.ingredient),
  );
  return [
    ...new Set(
      owners
        .filter((item) => item.accountId === viewer && !available.has(item.ingredient))
        .map((item) => item.ingredient),
    ),
  ];
}
