import { describe, expect, it } from "vitest";
import { foodSlug } from "./food-slug.ts";

describe("foodSlug", () => {
  it("joins lowercase words with dashes and drops punctuation", () => {
    expect(foodSlug("Lemon Water (Fancy)")).toBe("lemon-water-fancy");
    expect(foodSlug("  Greek yogurt ")).toBe("greek-yogurt");
  });

  it("files a single egg under the carton's photo", () => {
    expect(foodSlug("Egg")).toBe("eggs");
  });
});
