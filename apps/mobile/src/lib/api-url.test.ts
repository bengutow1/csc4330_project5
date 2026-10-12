import { describe, expect, it } from "vitest";
import { apiUrl } from "./api-url.ts";

describe("apiUrl", () => {
  it.each([
    { os: "ios", want: "http://localhost:8080" },
    { os: "android", want: "http://10.0.2.2:8080" },
    { os: "web", want: "/" },
  ])("points a $os build at the Mac's api", ({ os, want }) => {
    expect(apiUrl(os, undefined)).toBe(want);
  });

  // On a phone, localhost is the phone, so the LAN address has to be able to win everywhere.
  it.each(["ios", "android", "web"])("lets EXPO_PUBLIC_API_URL win on %s", (os) => {
    expect(apiUrl(os, "http://192.168.1.20:8080")).toBe("http://192.168.1.20:8080");
  });

  it("treats an empty override as unset", () => {
    expect(apiUrl("ios", "")).toBe("http://localhost:8080");
  });
});
