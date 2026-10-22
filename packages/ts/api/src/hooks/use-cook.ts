import type { DescService } from "@bufbuild/protobuf";
import { createConnectQueryKey, skipToken, useMutation, useQuery } from "@connectrpc/connect-query";
import type { Diet, Equipment } from "@fries-in-the-bag/apigen/account/v1/account_pb";
import { BoardService } from "@fries-in-the-bag/apigen/board/v1/board_pb";
import { CookService } from "@fries-in-the-bag/apigen/cook/v1/cook_pb";
import { type QueryClient, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef } from "react";

/** RECIPES is every recipe read, finite or infinite: the one key the cook family invalidates. */
export const RECIPES = createConnectQueryKey({ schema: CookService, cardinality: undefined });

// The services whose queries count plates and their points, so a new plate stales them: the
// board's Chefs. The profile (#30) joins it, and usePlateMeal picks each one up from here.
const PLATE_COUNTERS: readonly DescService[] = [BoardService];

/** PLATED is the family key of every query a plate changes, one per service that counts plates. */
export const PLATED = PLATE_COUNTERS.map((schema) =>
  createConnectQueryKey({ schema, cardinality: undefined }),
);

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
 * scan's last shelves (isPlaceholderData) until the new ones land. Another scan shows none, so its
 * fridge never wears the last one's recipes.
 */
export function useRecipes(scanId: bigint | undefined, filters: RecipeFilters) {
  // The scan whose shelves last landed, which only a finished read moves.
  const shownScan = useRef<bigint | undefined>(undefined);
  const query = useQuery(CookService.method.listRecipes, recipesInput(scanId, filters), {
    placeholderData: (previous) => shelvesPlaceholder(previous, shownScan.current, scanId),
  });
  useEffect(() => {
    if (query.data !== undefined && !query.isPlaceholderData) {
      shownScan.current = scanId;
    }
  }, [query.data, query.isPlaceholderData, scanId]);
  return query;
}

/**
 * shelvesPlaceholder is what the shelves show while new ones load: the last ones, but only when
 * they belong to the scan being asked about.
 */
export function shelvesPlaceholder<T>(
  previous: T | undefined,
  shownScan: bigint | undefined,
  scanId: bigint | undefined,
): T | undefined {
  return shownScan !== undefined && shownScan === scanId ? previous : undefined;
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
 * usePlateMeal sends a photo of the finished dish to be checked and paid. A plate the model did not
 * pass comes back as a receipt that says so, never as an error. A failed send is never retried on
 * its own: each try spends one of the player's checks, so the player taps again. Either way it
 * stales what counts plates, so new points show on the board's next look.
 */
export function usePlateMeal() {
  const invalidatePlated = useInvalidatePlated();
  return useMutation(CookService.method.plateMeal, { onSettled: () => invalidatePlated() });
}

/** useInvalidatePlated stales every query that counts plates or their points (PLATED). */
export function useInvalidatePlated() {
  const queryClient = useQueryClient();
  return useCallback(() => invalidatePlated(queryClient), [queryClient]);
}

/**
 * invalidatePlated stales PLATED without waiting for it. Those queries sit on other tabs and
 * refetch behind the write, so a receipt never waits on the board's pages.
 */
export function invalidatePlated(queryClient: QueryClient) {
  for (const queryKey of PLATED) {
    void queryClient.invalidateQueries({ queryKey });
  }
}

/**
 * useInvalidateRecipes marks every recipe read stale, after a write changed what the shelves hold
 * (a fixed list, a plate).
 */
export function useInvalidateRecipes() {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: RECIPES });
}
