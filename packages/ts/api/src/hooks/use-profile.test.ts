import { create } from "@bufbuild/protobuf";
import { createConnectQueryKey } from "@connectrpc/connect-query";
import {
  ListPlatesResponseSchema,
  ProfileService,
} from "@fries-in-the-bag/apigen/profile/v1/profile_pb";
import { ScanService } from "@fries-in-the-bag/apigen/scan/v1/scan_pb";
import { partialMatchKey } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import { createApiTransport } from "../transport.ts";
import { nextPlatesCursor, PROFILE } from "./use-profile.ts";

describe("nextPlatesCursor", () => {
  it("follows the server's cursor while there is more", () => {
    const page = create(ListPlatesResponseSchema, { nextCursor: "MTc5MjYwMjAwMDAwMDAwMDAwMDo0Mg" });
    expect(nextPlatesCursor(page)).toBe("MTc5MjYwMjAwMDAwMDAwMDAwMDo0Mg");
  });

  it("ends the grid at an empty cursor", () => {
    expect(nextPlatesCursor(create(ListPlatesResponseSchema, {}))).toBeUndefined();
  });
});

describe("PROFILE", () => {
  // Keys built the way connect-query builds them, so a family that missed a read would show here.
  const transport = createApiTransport({ baseUrl: "http://api.test.invalid" });

  it("covers the profile and every page of its plates, so one invalidation refreshes the Me tab", () => {
    const profile = createConnectQueryKey({
      schema: ProfileService.method.getProfile,
      input: {},
      transport,
      cardinality: "finite",
    });
    const plates = createConnectQueryKey({
      schema: ProfileService.method.listPlates,
      input: { cursor: "" },
      transport,
      cardinality: "infinite",
      pageParamKey: "cursor",
    });
    expect(partialMatchKey(profile, PROFILE)).toBe(true);
    expect(partialMatchKey(plates, PROFILE)).toBe(true);
  });

  it("leaves other services' reads alone", () => {
    const lastScan = createConnectQueryKey({
      schema: ScanService.method.getLastScan,
      input: {},
      transport,
      cardinality: "finite",
    });
    expect(partialMatchKey(lastScan, PROFILE)).toBe(false);
  });
});
