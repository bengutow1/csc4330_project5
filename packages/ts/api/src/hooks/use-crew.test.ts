import { create } from "@bufbuild/protobuf";
import { createConnectQueryKey } from "@connectrpc/connect-query";
import { Board, BoardService, Range } from "@fries-in-the-bag/apigen/board/v1/board_pb";
import { CrewService, ListMyCrewsResponseSchema } from "@fries-in-the-bag/apigen/crew/v1/crew_pb";
import {
  GetProfileResponseSchema,
  ProfileService,
} from "@fries-in-the-bag/apigen/profile/v1/profile_pb";
import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import { createApiTransport } from "../transport.ts";
import { invalidateCrews } from "./use-crew.ts";

const transport = createApiTransport({ baseUrl: "http://api.test.invalid" });

// Keys built the way the hooks build them: the crew list, a crew's board page and the profile.
const CREWS = createConnectQueryKey({
  schema: CrewService.method.listMyCrews,
  input: {},
  transport,
  cardinality: "finite",
});
const CREW_BOARD = createConnectQueryKey({
  schema: BoardService.method.getBoard,
  input: { board: Board.CHEFS, range: Range.THIS_WEEK, crewId: 7n, cursor: "" },
  transport,
  cardinality: "infinite",
});
const PROFILE = createConnectQueryKey({
  schema: ProfileService.method.getProfile,
  input: {},
  transport,
  cardinality: "finite",
});

describe("invalidateCrews", () => {
  it("stales the crew list and every board, crew boards included", () => {
    const client = new QueryClient();
    client.setQueryData(CREWS, create(ListMyCrewsResponseSchema, {}));
    client.setQueryData(CREW_BOARD, { pages: [], pageParams: [] });

    invalidateCrews(client);

    expect(client.getQueryState(CREWS)?.isInvalidated).toBe(true);
    expect(client.getQueryState(CREW_BOARD)?.isInvalidated).toBe(true);
  });

  it("leaves what a crew cannot change alone", () => {
    const client = new QueryClient();
    client.setQueryData(PROFILE, create(GetProfileResponseSchema, {}));

    invalidateCrews(client);

    expect(client.getQueryState(PROFILE)?.isInvalidated).toBe(false);
  });
});
