import type { DescService } from "@bufbuild/protobuf";
import {
  createConnectQueryKey,
  skipToken,
  useInfiniteQuery,
  useMutation,
} from "@connectrpc/connect-query";
import {
  FriesService,
  type ListSpotsResponse,
  type SpotFilter,
} from "@fries-in-the-bag/apigen/fries/v1/fries_pb";
import { ProfileService } from "@fries-in-the-bag/apigen/profile/v1/profile_pb";
import { keepPreviousData, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { useSettled } from "../use-settled.ts";

// The services whose queries count fry runs, so a new one stales them: the profile's fry runs and
// fry ratio. The board's Hall of Fries (#28) joins it, and useLogFryRun picks each one up from here.
const FRY_RUN_COUNTERS: readonly DescService[] = [ProfileService];

// How long typing must pause before the words become a request. Each request spends one of the
// player's ListSpots calls, so a search typed key by key must not spend one a key.
const SEARCH_SETTLE_MS = 300;

/** SpotsSearch is what the spots screen asks for: where the player is, what they typed, and which chips are on. */
export type SpotsSearch = {
  /** The player's point in degrees; undefined while their location is off. */
  at?: { lat: number; lng: number };
  /** Words to search by. With location off, they say where to look. */
  query?: string;
  /** The chips that are on, each at most once. */
  filters: readonly SpotFilter[];
};

/**
 * useSpots lists the spots for a search a page at a time, in the server's order and narrowed by
 * the server's filters. The words wait until typing pauses for SEARCH_SETTLE_MS; a point or a chip
 * applies at once. A new search keeps the last list on screen, marked isPlaceholderData, until
 * its own answer lands. Retry is off: every call spends a places request, and the player taps
 * again instead.
 */
export function useSpots(search: SpotsSearch) {
  const query = useSettled(search.query ?? "", SEARCH_SETTLE_MS);
  return useInfiniteQuery(FriesService.method.listSpots, spotsInput({ ...search, query }), {
    pageParamKey: "cursor",
    getNextPageParam: nextSpotsCursor,
    placeholderData: keepPreviousData,
    retry: false,
  });
}

/**
 * spotsInput is what useSpots sends for a search: skipToken until there is a point or words to
 * search by, since the server answers neither.
 */
export function spotsInput(search: SpotsSearch) {
  const query = search.query?.trim() ?? "";
  if (search.at === undefined && query === "") {
    return skipToken;
  }
  return {
    lat: search.at?.lat,
    lng: search.at?.lng,
    query,
    filters: [...search.filters],
    cursor: "",
  };
}

/** nextSpotsCursor is the cursor for the page after this one; undefined ends the list. */
export function nextSpotsCursor(page: ListSpotsResponse) {
  return page.nextCursor === "" ? undefined : page.nextCursor;
}

/** useLogFryRun logs that the player went to a spot. It earns no points; it stales what counts fry runs. */
export function useLogFryRun() {
  const invalidateFryRuns = useInvalidateFryRuns();
  return useMutation(FriesService.method.logFryRun, {
    onSettled: () => invalidateFryRuns(),
  });
}

/**
 * useInvalidateFryRuns stales every query that counts fry runs, so a new run shows wherever runs
 * are counted.
 */
export function useInvalidateFryRuns() {
  const queryClient = useQueryClient();
  return useCallback(
    () =>
      Promise.all(
        FRY_RUN_COUNTERS.map((schema) =>
          queryClient.invalidateQueries({
            queryKey: createConnectQueryKey({ schema, cardinality: undefined }),
          }),
        ),
      ),
    [queryClient],
  );
}
