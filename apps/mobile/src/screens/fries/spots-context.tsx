import { type SpotFilter, useSpots } from "@fries-in-the-bag/api";
import { usePathname } from "expo-router";
import { createContext, type ReactNode, useContext, useEffect, useState } from "react";
import { STARTING_FILTERS, sendableFilters } from "@/lib/spots";
import { usePlayerPoint } from "@/screens/fries/use-player-point";

function useSpotsSource() {
  const pathname = usePathname();
  const [openedList, setOpenedList] = useState(false);
  useEffect(() => {
    if (pathname === "/fries/spots") setOpenedList(true);
  }, [pathname]);
  const place = usePlayerPoint();
  const [filters, setFilters] = useState<readonly SpotFilter[]>(STARTING_FILTERS);
  const [query, setQuery] = useState("");
  const hasPoint = place.point !== undefined;
  const searching = hasPoint || query.trim() !== "";
  const sent = {
    at: place.point,
    query: hasPoint ? "" : query,
    filters: sendableFilters(filters, hasPoint),
  };
  // A direct link to roulette cannot start a search of its own.
  const spots = useSpots(openedList ? sent : { filters: [], query: "" });
  const pages = spots.data?.pages ?? [];
  const list = pages.flatMap((page) => page.spots);
  return {
    place,
    filters,
    setFilters,
    query,
    setQuery,
    hasPoint,
    searching,
    sent,
    spots,
    pages,
    list,
  };
}

const SpotsContext = createContext<ReturnType<typeof useSpotsSource> | undefined>(undefined);

/** Keeps F2 and roulette on one query without copying server data into navigation state. */
export function SpotsProvider({ children }: { children: ReactNode }) {
  const source = useSpotsSource();
  return <SpotsContext.Provider value={source}>{children}</SpotsContext.Provider>;
}

/** Reads the loaded spots and the same filters on both screens. */
export function useLoadedSpots() {
  const source = useContext(SpotsContext);
  if (source === undefined) {
    throw new Error("Spots screens need SpotsProvider");
  }
  return source;
}
