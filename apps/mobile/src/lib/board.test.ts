import { Board, Range } from "@fries-in-the-bag/api";
import { describe, expect, it } from "vitest";
import {
  BOARD_OPTIONS,
  boardItems,
  boardOf,
  distinctRows,
  emptyCopy,
  itemsInView,
  leaderLine,
  pinsMyRow,
  podiumOrder,
  podiumStep,
  RANGE_OPTIONS,
  rangeOf,
  rowLabel,
  subtitle,
} from "./board.ts";

// A row the way the server sends one: rank, handle, avatar, value and the crown.
function row(rank: number, handle: string, value: bigint, crown = false) {
  return { rank, handle, avatar: "🍟", value, crown };
}

describe("the switches", () => {
  it("list screen 7's boards and ranges in its order", () => {
    expect(BOARD_OPTIONS.map((option) => option.label)).toEqual(["Chefs", "Hall of Fries"]);
    expect(RANGE_OPTIONS.map((option) => option.label)).toEqual(["This week", "All time"]);
  });

  it("send each choice as its own request field", () => {
    expect(boardOf("chefs")).toBe(Board.CHEFS);
    expect(boardOf("hall_of_fries")).toBe(Board.HALL_OF_FRIES);
    expect(rangeOf("this_week")).toBe(Range.THIS_WEEK);
    expect(rangeOf("all_time")).toBe(Range.ALL_TIME);
  });
});

describe("subtitle", () => {
  it.each([
    { range: "this_week", want: "Campus · this week" },
    { range: "all_time", want: "Campus · all time" },
  ] as const)("is $want for $range", ({ range, want }) => {
    expect(subtitle(range)).toBe(want);
  });
});

describe("boardItems", () => {
  it("puts the first three rows on the podium and the rest below, in the server's order", () => {
    const rows = [
      row(1, "test_ann", 9n),
      row(1, "test_bob", 9n),
      row(3, "test_cat", 7n),
      row(4, "test_dan", 5n),
      row(5, "test_eli", 2n),
    ];

    const items = boardItems(rows);

    expect(items).toEqual([
      { kind: "podium", rows: rows.slice(0, 3) },
      { kind: "row", row: rows[3] },
      { kind: "row", row: rows[4] },
    ]);
  });

  it("is a podium alone when three or fewer have a value", () => {
    const rows = [row(1, "test_ann", 2n), row(2, "test_bob", 1n)];

    expect(boardItems(rows)).toEqual([{ kind: "podium", rows }]);
  });

  it("is nothing for an empty board, so the empty state shows", () => {
    expect(boardItems([])).toEqual([]);
  });
});

describe("podiumOrder", () => {
  it("stands the second row on the left, the first in the middle and the third on the right", () => {
    const [first, second, third] = [row(1, "a", 3n), row(2, "b", 2n), row(3, "c", 1n)];

    expect(podiumOrder([first, second, third])).toEqual([second, first, third]);
  });

  it("leaves a step empty when fewer than three are ranked", () => {
    const first = row(1, "a", 3n);
    const second = row(2, "b", 2n);

    expect(podiumOrder([first])).toEqual([undefined, first, undefined]);
    expect(podiumOrder([first, second])).toEqual([second, first, undefined]);
  });
});

describe("podiumStep", () => {
  it.each([
    { rank: 1, want: "first" },
    { rank: 2, want: "second" },
    { rank: 3, want: "third" },
    // A tie for second pushes the next rank to 4, and it still stands on the lowest step.
    { rank: 4, want: "third" },
  ] as const)("puts rank $rank on the $want step", ({ rank, want }) => {
    expect(podiumStep(rank)).toBe(want);
  });
});

describe("pinsMyRow", () => {
  const podium = {
    kind: "podium",
    rows: [row(1, "test_ann", 9n), row(2, "test_bob", 8n)],
  } as const;
  const dan = { kind: "row", row: row(4, "test_dan", 5n) } as const;

  it("pins my row when nothing on screen shows it", () => {
    expect(pinsMyRow("test_maya", [podium, dan])).toBe(true);
  });

  it("does not pin a row the list is showing", () => {
    expect(pinsMyRow("test_dan", [podium, dan])).toBe(false);
  });

  it("does not pin a row standing on the podium while the podium is on screen", () => {
    expect(pinsMyRow("test_bob", [podium, dan])).toBe(false);
  });

  it("pins a podium row once the podium has scrolled away", () => {
    expect(pinsMyRow("test_bob", [dan])).toBe(true);
  });

  it("pins nothing for a player with no value in the range", () => {
    expect(pinsMyRow(undefined, [dan])).toBe(false);
  });

  it("pins nothing before the list first says what is on screen", () => {
    expect(pinsMyRow("test_maya", undefined)).toBe(false);
  });
});

describe("itemsInView", () => {
  it("reads the positions in view from the board as it is now", () => {
    expect(itemsInView(["a", "b", "c", "d"], [1, 2])).toEqual(["b", "c"]);
  });

  it("leaves out a position past the end of a shorter board", () => {
    expect(itemsInView(["a", "b"], [1, 2])).toEqual(["b"]);
  });

  it("stays unknown until the list first says what is in view", () => {
    expect(itemsInView(["a"], undefined)).toBeUndefined();
  });

  it("pins my row after a switch moves it off the positions in view", () => {
    // On the old board maya stood fourth, in view. On the new one she is eighth, below it, and the
    // list need not say so again, since the same positions stay in view.
    const before = boardItems(
      ["test_ann", "test_bob", "test_cat", "test_maya"].map((h, i) => row(i + 1, h, 9n)),
    );
    const after = boardItems(
      [
        "test_ann",
        "test_bob",
        "test_cat",
        "test_dan",
        "test_eve",
        "test_fay",
        "test_gus",
        "test_maya",
      ].map((h, i) => row(i + 1, h, 9n)),
    );
    const inView = [0, 1];
    expect(pinsMyRow("test_maya", itemsInView(before, inView))).toBe(false);
    expect(pinsMyRow("test_maya", itemsInView(after, inView))).toBe(true);
  });
});

describe("distinctRows", () => {
  it("keeps each player's first row, in the server's order", () => {
    const rows = [row(1, "test_ann", 9n), row(2, "test_bob", 8n), row(5, "test_ann", 2n)];
    expect(distinctRows(rows)).toEqual([row(1, "test_ann", 9n), row(2, "test_bob", 8n)]);
  });
});

describe("rowLabel", () => {
  it("says the rank, whose row it is and the value with its unit", () => {
    expect(rowLabel("chefs", row(4, "ramen_king", 610n), false)).toBe(
      "Rank 4, ramen_king, 610 points",
    );
  });

  it("says one of a unit without a plural", () => {
    expect(rowLabel("chefs", row(9, "ramen_king", 1n), false)).toBe("Rank 9, ramen_king, 1 point");
    expect(rowLabel("hall_of_fries", row(2, "dorm_dan", 1n), false)).toBe(
      "Rank 2, dorm_dan, 1 fry run",
    );
  });

  it("calls the player's own row theirs", () => {
    expect(rowLabel("hall_of_fries", row(3, "test_maya", 5n), true)).toBe(
      "Rank 3, you, 5 fry runs",
    );
  });

  it("says when the row wears the crown", () => {
    expect(rowLabel("hall_of_fries", row(1, "dorm_dan", 1240n, true), false)).toBe(
      "Rank 1, dorm_dan, 1,240 fry runs, wearing the crown",
    );
  });
});

describe("leaderLine", () => {
  it("names this week's Hall of Fries leader, as screen 7 writes it", () => {
    expect(leaderLine(row(1, "dorm_fridge_dan", 5n))).toBe("dorm_fridge_dan leads with 5 fry runs");
  });

  it("says one fry run without a plural", () => {
    expect(leaderLine(row(1, "dorm_fridge_dan", 1n))).toBe("dorm_fridge_dan leads with 1 fry run");
  });
});

describe("emptyCopy", () => {
  it.each([
    {
      board: "chefs",
      range: "this_week",
      want: { title: "No chefs yet this week.", line: "Plate a meal and the top spot is yours." },
    },
    {
      board: "chefs",
      range: "all_time",
      want: { title: "No chefs yet.", line: "Plate a meal and the top spot is yours." },
    },
    {
      board: "hall_of_fries",
      range: "this_week",
      want: { title: "No fry runs yet this week.", line: "Every fridge has held up so far." },
    },
    {
      board: "hall_of_fries",
      range: "all_time",
      want: { title: "No fry runs yet.", line: "Every fridge has held up so far." },
    },
  ] as const)("names the $board board over $range", ({ board, range, want }) => {
    expect(emptyCopy(board, range)).toEqual(want);
  });
});
