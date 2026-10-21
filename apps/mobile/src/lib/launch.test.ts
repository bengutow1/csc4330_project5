import { describe, expect, it } from "vitest";
import { launchCoverShows } from "./launch.ts";

describe("launchCoverShows", () => {
  // Still asking, or the server can't be reached: neither is "no account", so onboarding waits.
  it.each([
    { route: "onboarding" },
    { route: "(tabs)" },
    { route: "+not-found" },
    { route: undefined },
  ])("covers $route until the app knows whose phone this is", ({ route }) => {
    expect(launchCoverShows({ signedIn: false, noSession: false, route })).toBe(true);
  });

  // /status has to be able to say the API is down, and the kit needs only Metro.
  it.each(["status", "kit"])("leaves %s open while the app can't tell", (route) => {
    expect(launchCoverShows({ signedIn: false, noSession: false, route })).toBe(false);
  });

  it.each([
    { what: "an account", signedIn: true, noSession: false },
    { what: "no session", signedIn: false, noSession: true },
  ])("lifts once the server answers with $what", ({ signedIn, noSession }) => {
    expect(launchCoverShows({ signedIn, noSession, route: "onboarding" })).toBe(false);
  });
});
