import { createConnectQueryKey, skipToken, useQuery } from "@connectrpc/connect-query";
import type { Diet, Equipment } from "@fries-in-the-bag/apigen/account/v1/account_pb";
import { CookService } from "@fries-in-the-bag/apigen/cook/v1/cook_pb";
import { keepPreviousData, useQueryClient } from "@tanstack/react-query";

/** RECIPES is every recipe read, finite or infinite: the one key the cook family invalidates. */
export const RECIPES = createConnectQueryKey({ schema: CookService, cardinality: undefined });

/** RecipeFilters are the chips over the shelves. The server applies every one of them. */
export type RecipeFilters = {
  /** The longest a recipe may take: 10, 20 or 45 minutes, or 0 for any. */
  maxMinutes: number;
  /** Keep the recipes that need nothing beyond this equipment; empty for any. */
  equipment: Equipment[];
  /** Keep the recipes that fit this diet; Diet.NONE for any. */
  diet: Diet;
};

/**
 * useRecipes reads the two shelves for one of the player's scans, narrowed by the filters. Each
 * filter is in the input, so each combination keeps its own answer, and a changed chip shows the
 * last shelves (isPlaceholderData) until the new ones land.
 */
export function useRecipes(scanId: bigint | undefined, filters: RecipeFilters) {
  return useQuery(CookService.method.listRecipes, recipesInput(scanId, filters), {
    placeholderData: keepPreviousData,
  });
}

/** recipesInput is what useRecipes sends: skipToken until the scan id is a real one. */
export function recipesInput(scanId: bigint | undefined, filters: RecipeFilters) {
  if (scanId === undefined || scanId <= 0n) {
    return skipToken;
  }
  return {
    scanId,
    maxMinutes: filters.maxMinutes,
    equipment: filters.equipment,
    diet: filters.diet,
  };
}

/** useRecipe reads one of the player's recipes, and waits until there is one to ask about. */
export function useRecipe(recipeId: bigint | undefined) {
  return useQuery(CookService.method.getRecipe, recipeInput(recipeId));
}

/** recipeInput is what useRecipe sends: skipToken until the id is a real one. */
export function recipeInput(recipeId: bigint | undefined) {
  return recipeId === undefined || recipeId <= 0n ? skipToken : { recipeId };
}

/**
 * useInvalidateRecipes marks every recipe read stale, after a write changed what the shelves hold
 * (a fixed list, a plate).
 */
export function useInvalidateRecipes() {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: RECIPES });
}
