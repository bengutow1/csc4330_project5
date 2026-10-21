import { SpotFilter } from "@fries-in-the-bag/api";
import { describe, expect, it } from "vitest";
import {
  clockText,
  directionsUrl,
  distanceText,
  emptyReason,
  metaLine,
  openLine,
  priceText,
  sendableFilters,
  snapPoint,
  toggleFilter,
  travelText,
} from "./spots.ts";

describe("distanceText", () => {
  it.each([
    { meters: 0, want: "" },
    { meters: 300, want: "300 m" },
    { meters: 999, want: "999 m" },
    { meters: 1000, want: "1 km" },
    { meters: 1200, want: "1.2 km" },
    { meters: 2349, want: "2.3 km" },
    { meters: 5000, want: "5 km" },
  ])("writes $meters m as $want", ({ meters, want }) => {
    expect(distanceText(meters)).toBe(want);
  });
});

describe("priceText", () => {
  it.each([
    { level: 0, want: "" },
    { level: 1, want: "$" },
    { level: 2, want: "$$" },
    { level: 4, want: "$$$$" },
  ])("writes price level $level as $want", ({ level, want }) => {
    expect(priceText(level)).toBe(want);
  });
});

describe("metaLine", () => {
  it("is cuisine, price and distance, as screen 9 writes them", () => {
    expect(metaLine({ category: "Mexican", priceLevel: 1, distanceMeters: 2900 })).toBe(
      "Mexican · $ · 2.9 km",
    );
  });

  it("leaves out what the spot does not know", () => {
    expect(metaLine({ category: "", priceLevel: 2, distanceMeters: 0 })).toBe("$$");
    expect(metaLine({ category: "", priceLevel: 0, distanceMeters: 0 })).toBe("");
  });
});

describe("clockText", () => {
  it.each([
    { at: "02:00", want: "2 AM" },
    { at: "00:00", want: "midnight" },
    { at: "00:30", want: "12:30 AM" },
    { at: "12:00", want: "noon" },
    { at: "12:30", want: "12:30 PM" },
    { at: "14:30", want: "2:30 PM" },
    { at: "21:00", want: "9 PM" },
    { at: "", want: "" },
  ])("reads $at as $want", ({ at, want }) => {
    expect(clockText(at)).toBe(want);
  });
});

describe("openLine", () => {
  it("says until when an open spot is open", () => {
    expect(openLine({ openNow: true, closesAt: "02:00" })).toEqual({
      text: "Open until 2 AM",
      open: true,
    });
    expect(openLine({ openNow: true, closesAt: "00:00" })).toEqual({
      text: "Open until midnight",
      open: true,
    });
  });

  it("says open with no time when there is none to show", () => {
    expect(openLine({ openNow: true, closesAt: "" })).toEqual({ text: "Open now", open: true });
  });

  it("says closed, whatever the time", () => {
    expect(openLine({ openNow: false, closesAt: "" })).toEqual({ text: "Closed now", open: false });
  });
});

describe("travelText", () => {
  // Screen 9's own numbers: 2.3, 2.9 and 3.4 km by car are 6, 8 and 9 min; 1.1 km on foot is 14.
  it.each([
    { meters: 2300, walkable: false, want: "6 min" },
    { meters: 2900, walkable: false, want: "8 min" },
    { meters: 3400, walkable: false, want: "9 min" },
    { meters: 1100, walkable: true, want: "14 min walk" },
    { meters: 40, walkable: true, want: "1 min walk" },
  ])("puts $meters m at $want", ({ meters, walkable, want }) => {
    expect(travelText({ distanceMeters: meters, walkable })).toBe(want);
  });

  it("says nothing without a distance", () => {
    expect(travelText({ distanceMeters: 0, walkable: false })).toBe("");
  });
});

describe("directionsUrl", () => {
  const spot = { lat: 30.42, lng: -91.17, walkable: false };

  it("hands iOS to Apple Maps and Android to Google Maps", () => {
    expect(directionsUrl(spot, "ios")).toBe("https://maps.apple.com/?daddr=30.42,-91.17&dirflg=d");
    expect(directionsUrl(spot, "android")).toBe(
      "https://www.google.com/maps/dir/?api=1&destination=30.42,-91.17&travelmode=driving",
    );
  });

  it("asks for walking directions to a walkable spot", () => {
    const near = { ...spot, walkable: true };
    expect(directionsUrl(near, "ios")).toBe("https://maps.apple.com/?daddr=30.42,-91.17&dirflg=w");
    expect(directionsUrl(near, "android")).toBe(
      "https://www.google.com/maps/dir/?api=1&destination=30.42,-91.17&travelmode=walking",
    );
  });
});

describe("toggleFilter", () => {
  it("turns a chip on in the chips' order, whatever order they were tapped in", () => {
    expect(toggleFilter([SpotFilter.CHEAP], SpotFilter.OPEN_NOW)).toEqual([
      SpotFilter.OPEN_NOW,
      SpotFilter.CHEAP,
    ]);
  });

  it("turns a chip off", () => {
    expect(toggleFilter([SpotFilter.OPEN_NOW, SpotFilter.CHEAP], SpotFilter.OPEN_NOW)).toEqual([
      SpotFilter.CHEAP,
    ]);
  });

  it("leaves the list it was given alone", () => {
    const on = [SpotFilter.OPEN_NOW];
    toggleFilter(on, SpotFilter.CHEAP);
    expect(on).toEqual([SpotFilter.OPEN_NOW]);
  });
});

describe("sendableFilters", () => {
  const all = [SpotFilter.OPEN_NOW, SpotFilter.WALKABLE];

  it("sends Walkable only with a point, since the server refuses it without one", () => {
    expect(sendableFilters(all, true)).toEqual(all);
    expect(sendableFilters(all, false)).toEqual([SpotFilter.OPEN_NOW]);
  });
});

describe("emptyReason", () => {
  it("blames the hour when Open now is on", () => {
    expect(emptyReason([SpotFilter.OPEN_NOW, SpotFilter.CHEAP])).toBe("closed");
  });

  it("blames the chips otherwise", () => {
    expect(emptyReason([SpotFilter.CHEAP])).toBe("filtered");
    expect(emptyReason([])).toBe("filtered");
  });
});

describe("snapPoint", () => {
  it("rounds to four decimals, about 11 m, so a step between screens asks the same question", () => {
    expect(snapPoint(30.413349, -91.180049)).toEqual({ lat: 30.4133, lng: -91.18 });
    expect(snapPoint(30.41336, -91.18006)).toEqual({ lat: 30.4134, lng: -91.1801 });
  });
});
