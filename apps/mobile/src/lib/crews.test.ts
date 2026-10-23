import { describe, expect, it } from "vitest";
import { CAMPUS, memberLine, scopeOf } from "./crews.ts";

const DORM = { crewId: 7n, memberCount: 3, code: "123456" };
const LAB = { crewId: 9n, memberCount: 1, code: "654321" };

describe("scopeOf", () => {
  it("shows the crew the player picked while they are in it", () => {
    expect(scopeOf([DORM, LAB], 9n)).toBe(9n);
  });

  it("falls back to the campus once the picked crew is gone from their list", () => {
    expect(scopeOf([DORM], 9n)).toBe(CAMPUS);
  });

  it("shows the campus until the list has loaded, rather than a crew it cannot check", () => {
    expect(scopeOf(undefined, 7n)).toBe(CAMPUS);
  });

  it("keeps the campus when the campus is picked", () => {
    expect(scopeOf([DORM, LAB], CAMPUS)).toBe(CAMPUS);
  });
});

describe("memberLine", () => {
  it("counts the members", () => {
    expect(memberLine(DORM)).toBe("3 members");
  });

  it("says member for a crew of one", () => {
    expect(memberLine(LAB)).toBe("1 member");
  });
});
