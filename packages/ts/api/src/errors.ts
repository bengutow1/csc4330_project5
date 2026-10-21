import { Code, ConnectError } from "@connectrpc/connect";
import { BadRequestSchema } from "@fries-in-the-bag/apigen/error/v1/error_pb";

const GENERIC_MESSAGE = "Something went wrong. Try again.";

/**
 * errorMessage is the client twin of the server's handlers.ConnectErr. The codes whose text the
 * server wrote for a person are shown as sent; anything else was redacted on purpose, so it gets a
 * generic line.
 */
export function errorMessage(err: unknown): string {
  if (err instanceof ConnectError) {
    switch (err.code) {
      case Code.Unauthenticated:
      case Code.PermissionDenied:
      case Code.InvalidArgument:
      case Code.NotFound:
      case Code.AlreadyExists:
      case Code.FailedPrecondition:
      case Code.ResourceExhausted:
        return err.rawMessage;
      default:
        return GENERIC_MESSAGE;
    }
  }
  return GENERIC_MESSAGE;
}

/**
 * fieldErrors reads the error.v1.BadRequest detail as proto field name to message. Inputs are
 * named after those fields, so a form puts each message under its input with no mapping table. An
 * empty record means this was not a validation failure: fall back to errorMessage.
 */
export function fieldErrors(err: unknown): Record<string, string> {
  if (!(err instanceof ConnectError)) {
    return {};
  }
  const out: Record<string, string> = {};
  for (const detail of err.findDetails(BadRequestSchema)) {
    for (const violation of detail.fieldViolations) {
      out[violation.field] = violation.description;
    }
  }
  return out;
}

/** isUnauthenticated: no session, or a dead one. The app answers with onboarding, not a toast. */
export function isUnauthenticated(err: unknown): boolean {
  return err instanceof ConnectError && err.code === Code.Unauthenticated;
}

/**
 * isNotFound: the thing named does not exist, as distinct from a request that never got there.
 * A screen that only checks isError renders both the same way, and they mean opposite things.
 */
export function isNotFound(err: unknown): boolean {
  return err instanceof ConnectError && err.code === Code.NotFound;
}

/** isFailedPrecondition: the request was fine and the world has moved on (a combo that ended). */
export function isFailedPrecondition(err: unknown): boolean {
  return err instanceof ConnectError && err.code === Code.FailedPrecondition;
}

/**
 * isUnavailable: something answered and said it cannot serve right now, like a failed readiness
 * check. A server that was never reached is not this: connect reports that as Unknown.
 */
export function isUnavailable(err: unknown): boolean {
  return err instanceof ConnectError && err.code === Code.Unavailable;
}

/**
 * isUnreachable: the request never got an answer, because the phone is offline or the api is
 * down. A screen can say so in its own words ("Scans need a connection."), where errorMessage
 * only has the generic line. The api never sends Unknown itself: handlers.ConnectErr turns an
 * error it does not know into internal.
 */
export function isUnreachable(err: unknown): boolean {
  return err instanceof ConnectError && err.code === Code.Unknown;
}
