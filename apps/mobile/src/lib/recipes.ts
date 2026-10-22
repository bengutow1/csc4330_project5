import {
  Diet,
  Difficulty,
  type Equipment,
  type Recipe,
  type RecipeFilters,
} from "@fries-in-the-bag/api";
import { type Choice, DIETS, EQUIPMENT, toggled } from "./kitchen.ts";

/** TIME_LIMITS are the time chips, in minutes: the server keeps recipes that take no longer. */
export const TIME_LIMITS: readonly number[] = [10, 20, 45];

/** NO_FILTERS is every chip off, which the shelves open with. A module constant, so the key holds. */
export const NO_FILTERS: RecipeFilters = { maxMinutes: 0, equipment: [], diet: Diet.NONE };

/** DIET_CHIPS are the diet chips: every diet but none, which is every chip off. */
export const DIET_CHIPS: readonly Choice<Diet>[] = DIETS.filter((c) => c.value !== Diet.NONE);

/** withTime is filters with a time chip tapped. A recipe has one time, so one chip is on at most. */
export function withTime(filters: RecipeFilters, minutes: number): RecipeFilters {
  return { ...filters, maxMinutes: filters.maxMinutes === minutes ? 0 : minutes };
}

/**
 * withEquipment is filters with a piece of equipment tapped on or off. The list keeps the closed
 * set's order, so the same chips in any tapping order are the same query and its cached answer.
 */
export function withEquipment(filters: RecipeFilters, equipment: Equipment): RecipeFilters {
  const on = toggled(filters.equipment, equipment);
  return {
    ...filters,
    equipment: EQUIPMENT.map((c) => c.value).filter((value) => on.includes(value)),
  };
}

/** withDiet is filters with a diet chip tapped: that diet, or none again if it was on. */
export function withDiet(filters: RecipeFilters, diet: Diet): RecipeFilters {
  return { ...filters, diet: filters.diet === diet ? Diet.NONE : diet };
}

/** filtering says whether any chip is on. */
export function filtering(filters: RecipeFilters): boolean {
  return filters.maxMinutes !== 0 || filters.equipment.length > 0 || filters.diet !== Diet.NONE;
}

/**
 * emptyLine is the line under "Nothing you can make right now". With a chip on, turning it off
 * may find a meal, so it says so instead of blaming the list.
 */
export function emptyLine(filters: RecipeFilters): string {
  return filtering(filters)
    ? "Nothing fits these filters. Turn one off, or get fries."
    : "This list doesn't add up to a meal.";
}

/** haveLabel counts what the player has of everything a recipe takes ("Have 6 of 7"). */
export function haveLabel(recipe: Pick<Recipe, "uses" | "missing">): string {
  return `Have ${recipe.uses.length} of ${recipe.uses.length + recipe.missing.length}`;
}

/** minutesLabel is a recipe's whole time to the plate ("25 min"). */
export function minutesLabel(minutes: number): string {
  return `${minutes} min`;
}

// A difficulty added to the wire fails to compile here until it has a name.
const DIFFICULTIES: Record<Exclude<Difficulty, Difficulty.UNSPECIFIED>, string> = {
  [Difficulty.EASY]: "Easy",
  [Difficulty.MEDIUM]: "Medium",
  [Difficulty.HARD]: "Hard",
};

/** difficultyLabel names a difficulty, or none for one the app doesn't know. Proto enums are open. */
export function difficultyLabel(difficulty: Difficulty): string | undefined {
  const names: Partial<Record<Difficulty, string>> = DIFFICULTIES;
  return names[difficulty];
}

/**
 * equipmentLabel names what a recipe cooks with, in the server's order ("Oven, air fryer"), or
 * says it needs none. Equipment the app doesn't know is left out, and only it leaves nothing to
 * say: an empty list is the server saying none is needed.
 */
export function equipmentLabel(equipment: readonly Equipment[]): string | undefined {
  if (equipment.length === 0) {
    return "No equipment";
  }
  const names = equipment.flatMap((value) => {
    const label = EQUIPMENT.find((c) => c.value === value)?.label;
    return label === undefined ? [] : [label];
  });
  if (names.length === 0) {
    return undefined;
  }
  return names.map((name, i) => (i === 0 ? name : name.toLowerCase())).join(", ");
}

/** pointsLabel is the points plating a recipe would earn, as the server previewed them. */
export function pointsLabel(points: number): string {
  return `+${points} pts`;
}

/** stepsLabel counts a recipe's steps ("4 steps"). */
export function stepsLabel(count: number): string {
  return `${count} step${count === 1 ? "" : "s"}`;
}

/** recipeLabel is a recipe card read out whole, since a screen reader hears it as one button. */
export function recipeLabel(
  recipe: Pick<
    Recipe,
    "title" | "minutes" | "difficulty" | "equipment" | "uses" | "missing" | "pointsPreview"
  >,
): string {
  return [
    recipe.title,
    `${recipe.minutes} minute${recipe.minutes === 1 ? "" : "s"}`,
    difficultyLabel(recipe.difficulty),
    equipmentLabel(recipe.equipment),
    haveLabel(recipe),
    `${recipe.pointsPreview} points`,
  ]
    .filter((part) => part !== undefined)
    .join(", ");
}

/**
 * numbered gives each line of a recipe's list its place, counted from one: the step's number, and
 * the key a repeated line still differs by.
 */
export function numbered(list: readonly string[]): { n: number; text: string }[] {
  return list.map((text, i) => ({ n: i + 1, text }));
}

/**
 * parseRecipeId reads a recipe's id from its route: a whole number above zero, or undefined for
 * anything else, so the screen never asks the api about a recipe that can't exist.
 */
export function parseRecipeId(param: string | undefined): bigint | undefined {
  if (param === undefined || !/^\d+$/.test(param)) {
    return undefined;
  }
  const id = BigInt(param);
  return id > 0n ? id : undefined;
}
