import { createConnectQueryKey, useInfiniteQuery, useQuery } from "@connectrpc/connect-query";
import {
  type ListPlatesResponse,
  ProfileService,
} from "@fries-in-the-bag/apigen/profile/v1/profile_pb";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

// GetProfile takes no input, and ListPlates starts at its first page. Module constants keep the
// query keys the same on every render.
const PROFILE_INPUT = {};
const PLATES_INPUT = { cursor: "" };

/** PROFILE is every profile read, the profile and its plates: the one key the family invalidates. */
export const PROFILE = createConnectQueryKey({ schema: ProfileService, cardinality: undefined });

/** useProfile reads the caller's rank, stats and every badge, all decided by the server. */
export function useProfile() {
  return useQuery(ProfileService.method.getProfile, PROFILE_INPUT);
}

/** usePlates reads the caller's plates a page at a time, newest first, each with its photo. */
export function usePlates() {
  return useInfiniteQuery(ProfileService.method.listPlates, PLATES_INPUT, {
    pageParamKey: "cursor",
    getNextPageParam: nextPlatesCursor,
  });
}

/** nextPlatesCursor is the cursor for the page after this one; undefined ends the grid. */
export function nextPlatesCursor(page: ListPlatesResponse) {
  return page.nextCursor === "" ? undefined : page.nextCursor;
}

/** useInvalidateProfile marks every profile read stale, after a write changed what it shows. */
export function useInvalidateProfile() {
  const queryClient = useQueryClient();
  return useCallback(() => queryClient.invalidateQueries({ queryKey: PROFILE }), [queryClient]);
}
