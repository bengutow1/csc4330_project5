import { describe, expect, it } from "vitest";
import { lastWriteError } from "./last-write";

const saveFailed = new Error("save failed");
const deleteFailed = new Error("delete failed");

describe("lastWriteError", () => {
  // Settings saves and deletes. A delete that failed after an earlier failed save has to say why
  // the delete failed, not repeat the save's old message.
  it("speaks for a failed delete tried after a failed save", () => {
    const save = { error: saveFailed, submittedAt: 1_000 };
    const remove = { error: deleteFailed, submittedAt: 2_000 };
    expect(lastWriteError(save, remove)).toBe(deleteFailed);
  });

  it("speaks for a failed save tried after a failed delete", () => {
    const save = { error: saveFailed, submittedAt: 3_000 };
    const remove = { error: deleteFailed, submittedAt: 2_000 };
    expect(lastWriteError(save, remove)).toBe(saveFailed);
  });

  it("says nothing while the newest write is still out, whatever failed before it", () => {
    const save = { error: saveFailed, submittedAt: 1_000 };
    const remove = { error: null, submittedAt: 2_000 };
    expect(lastWriteError(save, remove)).toBeNull();
  });

  it("says nothing before any write is tried", () => {
    const untried = { error: null, submittedAt: 0 };
    expect(lastWriteError(untried, untried)).toBeNull();
  });
});
