import { QueryObserver } from "@tanstack/react-query";
import { afterEach, describe, expect, it, vi } from "vitest";
import { type AppStateEvents, wireAppState } from "./focus.ts";
import { makeQueryClient } from "./query-client.ts";

// A stand-in for React Native's AppState: it keeps the listener the data layer adds, so the test
// can say when the app leaves the foreground and comes back.
function fakeAppState() {
  let listener: ((status: string) => void) | undefined;
  const events: AppStateEvents = {
    addEventListener: (_type, next) => {
      listener = next;
      return {
        remove: () => {
          listener = undefined;
        },
      };
    },
  };
  return { events, emit: (status: string) => listener?.(status) };
}

// Counts the calls to one query's fetcher while an observer keeps the query mounted, as a screen
// would.
function watchQuery(client: ReturnType<typeof makeQueryClient>, name: string) {
  const counter = { fetches: 0, stop: () => {} };
  const observer = new QueryObserver(client, {
    queryKey: [name],
    queryFn: () => {
      counter.fetches += 1;
      return counter.fetches;
    },
  });
  counter.stop = observer.subscribe(() => {});
  return counter;
}

describe("wireAppState", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  // Prevents a screen showing stale data after the phone wakes: React Native has no window focus,
  // so without the wiring refetchOnWindowFocus never fires.
  it("refetches a stale query, and only a stale one, when the app becomes active", async () => {
    vi.useFakeTimers({ toFake: ["Date"] });
    const start = Date.now();
    const appState = fakeAppState();
    wireAppState(appState.events);
    const client = makeQueryClient();
    client.mount();

    // Wait for each answer to land in the cache, not just for the fetch to start: the time it lands
    // is the time its staleness counts from.
    const old = watchQuery(client, "old");
    await vi.waitFor(() => expect(client.getQueryData(["old"])).toBe(1));
    vi.setSystemTime(start + 50_000);
    const fresh = watchQuery(client, "fresh");
    await vi.waitFor(() => expect(client.getQueryData(["fresh"])).toBe(1));

    // A minute and a second after the first fetch: past the client's 60 s staleTime for "old",
    // eleven seconds into it for "fresh".
    vi.setSystemTime(start + 61_000);
    appState.emit("background");
    appState.emit("active");

    // One focus event refetches every stale query in a single pass, so once "old" has its second
    // answer, "fresh" has already been passed over.
    await vi.waitFor(() => expect(client.getQueryData(["old"])).toBe(2));
    expect(old.fetches).toBe(2);
    expect(fresh.fetches).toBe(1);

    old.stop();
    fresh.stop();
    client.unmount();
    client.clear();
  });
});
