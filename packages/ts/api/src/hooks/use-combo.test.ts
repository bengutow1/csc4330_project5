import { create } from "@bufbuild/protobuf";
import { skipToken } from "@connectrpc/connect-query";
import {
  CombinedVerdictSchema,
  ComboSchema,
  ComboState,
} from "@fries-in-the-bag/apigen/combo/v1/combo_pb";
import { describe, expect, it } from "vitest";
import { comboInput, comboWaiting } from "./use-combo.ts";

describe("comboWaiting", () => {
  it("polls an open combo until its combined verdict lands", () => {
    expect(comboWaiting(create(ComboSchema, { state: ComboState.OPEN }))).toBe(true);
  });

  it("stops once both fridges are in", () => {
    const revealed = create(ComboSchema, {
      state: ComboState.OPEN,
      verdict: create(CombinedVerdictSchema, { score: 62 }),
    });
    expect(comboWaiting(revealed)).toBe(false);
  });

  it("stops once the host ends it, verdict or not", () => {
    expect(comboWaiting(create(ComboSchema, { state: ComboState.ENDED }))).toBe(false);
  });

  it("does not poll before anything has loaded, so a failed read is not hammered", () => {
    expect(comboWaiting(undefined)).toBe(false);
  });
});

describe("comboInput", () => {
  it("asks for a real id", () => {
    expect(comboInput(7n)).toEqual({ comboId: 7n });
  });

  it("sends nothing for a missing or impossible id", () => {
    expect(comboInput(undefined)).toBe(skipToken);
    expect(comboInput(0n)).toBe(skipToken);
  });
});
