import { useQuery } from "@connectrpc/connect-query";
import { HealthService } from "@fries-in-the-bag/apigen/health/v1/health_pb";

// Ready takes no input. A module constant keeps the query key the same on every render.
const READY_INPUT = {};

/**
 * useReady asks the api whether it can serve: up, and able to reach its database. Retry is off
 * because a failing probe is the answer the screen asked for, not a blip to hide.
 */
export function useReady() {
  return useQuery(HealthService.method.ready, READY_INPUT, { retry: false });
}
