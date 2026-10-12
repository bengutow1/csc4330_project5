import { Code, ConnectError } from "@connectrpc/connect";
import { describe, expect, it } from "vitest";
import {
  errorMessage,
  fieldErrors,
  isFailedPrecondition,
  isNotFound,
  isUnauthenticated,
  isUnavailable,
} from "./errors.ts";

const GENERIC = "Something went wrong. Try again.";

describe("errorMessage", () => {
  // Mirrors the arms of handlers.ConnectErr whose text is written for a person.
  const written: Code[] = [
    Code.Unauthenticated,
    Code.PermissionDenied,
    Code.InvalidArgument,
    Code.NotFound,
    Code.AlreadyExists,
    Code.FailedPrecondition,
    Code.ResourceExhausted,
  ];
  const redacted: Code[] = [Code.Internal, Code.Unknown, Code.Unavailable, Code.DeadlineExceeded];

  it.each(written)("shows the server's own words for code %s", (code) => {
    expect(errorMessage(new ConnectError("that combo is full", code))).toBe("that combo is full");
  });

  it.each(redacted)("says something generic for code %s, whatever the server wrote", (code) => {
    expect(errorMessage(new ConnectError("dial tcp 10.0.0.4:5432", code))).toBe(GENERIC);
  });

  it("says something generic for an error that did not come from the api", () => {
    expect(errorMessage(new Error("undefined is not a function"))).toBe(GENERIC);
    expect(errorMessage("a thrown string")).toBe(GENERIC);
  });
});

describe("fieldErrors", () => {
  it("is empty when there is nothing to put under an input", () => {
    expect(fieldErrors(new Error("boom"))).toEqual({});
    expect(fieldErrors(new ConnectError("that combo is full", Code.ResourceExhausted))).toEqual({});
  });
});

describe("predicates", () => {
  const cases = [
    { name: "isUnauthenticated", is: isUnauthenticated, code: Code.Unauthenticated },
    { name: "isNotFound", is: isNotFound, code: Code.NotFound },
    { name: "isFailedPrecondition", is: isFailedPrecondition, code: Code.FailedPrecondition },
    { name: "isUnavailable", is: isUnavailable, code: Code.Unavailable },
  ];

  it.each(cases)("$name matches its own code and nothing else", ({ is, code }) => {
    expect(is(new ConnectError("x", code))).toBe(true);
    expect(is(new ConnectError("x", Code.Internal))).toBe(false);
    expect(is(new Error("x"))).toBe(false);
  });
});
