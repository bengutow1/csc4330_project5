import { Allergy, Diet, Equipment, RoastLevel, Staple } from "@fries-in-the-bag/api";

/** Choice is one value of a kitchen's closed set and the words the app shows for it. */
export interface Choice<T> {
  value: T;
  label: string;
}

// Every list below runs in its closed set's order (docs/contracts.md), which is the order the
// server stores and returns.

/** EQUIPMENT is what a kitchen can cook with. */
export const EQUIPMENT: readonly Choice<Equipment>[] = [
  { value: Equipment.MICROWAVE, label: "Microwave" },
  { value: Equipment.STOVETOP, label: "Stovetop" },
  { value: Equipment.OVEN, label: "Oven" },
  { value: Equipment.AIR_FRYER, label: "Air fryer" },
];

/** DIETS are the ways of eating recipes respect. None means no rules. */
export const DIETS: readonly Choice<Diet>[] = [
  { value: Diet.NONE, label: "Anything" },
  { value: Diet.VEGETARIAN, label: "Vegetarian" },
  { value: Diet.VEGAN, label: "Vegan" },
  { value: Diet.PESCATARIAN, label: "Pescatarian" },
];

/** ALLERGIES are hard filters: a recipe with one never reaches the player. */
export const ALLERGIES: readonly Choice<Allergy>[] = [
  { value: Allergy.PEANUTS, label: "Peanuts" },
  { value: Allergy.TREE_NUTS, label: "Tree nuts" },
  { value: Allergy.MILK, label: "Milk" },
  { value: Allergy.EGGS, label: "Eggs" },
  { value: Allergy.WHEAT, label: "Wheat" },
  { value: Allergy.SOY, label: "Soy" },
  { value: Allergy.FISH, label: "Fish" },
  { value: Allergy.SHELLFISH, label: "Shellfish" },
  { value: Allergy.SESAME, label: "Sesame" },
];

/** STAPLES are the basics set once here and never scanned. A new account has all of them. */
export const STAPLES: readonly Choice<Staple>[] = [
  { value: Staple.SALT, label: "Salt" },
  { value: Staple.PEPPER, label: "Pepper" },
  { value: Staple.OIL, label: "Oil" },
  { value: Staple.BUTTER, label: "Butter" },
  { value: Staple.RICE, label: "Rice" },
  { value: Staple.PASTA, label: "Pasta" },
  { value: Staple.FLOUR, label: "Flour" },
  { value: Staple.SUGAR, label: "Sugar" },
  { value: Staple.SPICES, label: "Spices" },
];

/**
 * ROAST_LEVELS are how hard the roast hits, each with a sample about the same fridge, so the
 * player hears the difference. Roasts target the fridge, never the person.
 */
export const ROAST_LEVELS: readonly (Choice<RoastLevel> & { sample: string })[] = [
  { value: RoastLevel.MILD, label: "Mild", sample: "Some eggs and cheese. That's a start." },
  {
    value: RoastLevel.MEDIUM,
    label: "Medium",
    sample: "Eggs, cheese and a lot of shelf. We can work with this.",
  },
  {
    value: RoastLevel.EXTRA_CRISPY,
    label: "Extra crispy",
    sample: "Two eggs and a cheese single. This fridge echoes.",
  },
];

/** toggled is list with value added, or taken out if it was in. */
export function toggled<T>(list: readonly T[], value: T): T[] {
  return list.includes(value) ? list.filter((v) => v !== value) : [...list, value];
}
