import { Category } from "@fries-in-the-bag/api";

/** Item is one ingredient on a list: a verdict's, or the one the edit sheet is building. */
export interface Item {
  name: string;
  category: Category;
  quantity: string;
}

/** A common ingredient the edit sheet can suggest, with the category it goes under. */
export interface Common {
  name: string;
  category: Category;
}

const MAX_SUGGESTIONS = 5;

// What the edit sheet suggests while the player types, spelled as the screen shows them. No
// staples: those are set once in the profile and never scanned (docs/contracts.md).
export const COMMON: readonly Common[] = [
  { name: "Chicken", category: Category.PROTEIN },
  { name: "Chicken breast", category: Category.PROTEIN },
  { name: "Ground beef", category: Category.PROTEIN },
  { name: "Steak", category: Category.PROTEIN },
  { name: "Bacon", category: Category.PROTEIN },
  { name: "Ham", category: Category.PROTEIN },
  { name: "Sausage", category: Category.PROTEIN },
  { name: "Pepperoni", category: Category.PROTEIN },
  { name: "Turkey", category: Category.PROTEIN },
  { name: "Salmon", category: Category.PROTEIN },
  { name: "Tuna", category: Category.PROTEIN },
  { name: "Shrimp", category: Category.PROTEIN },
  { name: "Tofu", category: Category.PROTEIN },
  { name: "Eggs", category: Category.PROTEIN },
  { name: "Black beans", category: Category.PROTEIN },
  { name: "Chickpeas", category: Category.PROTEIN },
  { name: "Lentils", category: Category.PROTEIN },
  { name: "Peanut butter", category: Category.PROTEIN },
  { name: "Spinach", category: Category.VEGETABLE },
  { name: "Lettuce", category: Category.VEGETABLE },
  { name: "Kale", category: Category.VEGETABLE },
  { name: "Broccoli", category: Category.VEGETABLE },
  { name: "Carrots", category: Category.VEGETABLE },
  { name: "Celery", category: Category.VEGETABLE },
  { name: "Cucumber", category: Category.VEGETABLE },
  { name: "Tomatoes", category: Category.VEGETABLE },
  { name: "Bell peppers", category: Category.VEGETABLE },
  { name: "Onion", category: Category.VEGETABLE },
  { name: "Garlic", category: Category.VEGETABLE },
  { name: "Potatoes", category: Category.VEGETABLE },
  { name: "Sweet potatoes", category: Category.VEGETABLE },
  { name: "Mushrooms", category: Category.VEGETABLE },
  { name: "Zucchini", category: Category.VEGETABLE },
  { name: "Corn", category: Category.VEGETABLE },
  { name: "Frozen peas", category: Category.VEGETABLE },
  { name: "Green beans", category: Category.VEGETABLE },
  { name: "Cabbage", category: Category.VEGETABLE },
  { name: "Jalapeños", category: Category.VEGETABLE },
  { name: "Apples", category: Category.FRUIT },
  { name: "Bananas", category: Category.FRUIT },
  { name: "Oranges", category: Category.FRUIT },
  { name: "Lemons", category: Category.FRUIT },
  { name: "Limes", category: Category.FRUIT },
  { name: "Strawberries", category: Category.FRUIT },
  { name: "Blueberries", category: Category.FRUIT },
  { name: "Grapes", category: Category.FRUIT },
  { name: "Avocado", category: Category.FRUIT },
  { name: "Pineapple", category: Category.FRUIT },
  { name: "Mango", category: Category.FRUIT },
  { name: "Bread", category: Category.GRAIN },
  { name: "Tortillas", category: Category.GRAIN },
  { name: "Bagels", category: Category.GRAIN },
  { name: "Oats", category: Category.GRAIN },
  { name: "Cereal", category: Category.GRAIN },
  { name: "Quinoa", category: Category.GRAIN },
  { name: "Noodles", category: Category.GRAIN },
  { name: "Instant ramen", category: Category.GRAIN },
  { name: "Milk", category: Category.DAIRY },
  { name: "Cheddar", category: Category.DAIRY },
  { name: "Mozzarella", category: Category.DAIRY },
  { name: "Parmesan", category: Category.DAIRY },
  { name: "Shredded cheese", category: Category.DAIRY },
  { name: "Cream cheese", category: Category.DAIRY },
  { name: "Greek yogurt", category: Category.DAIRY },
  { name: "Sour cream", category: Category.DAIRY },
  { name: "Heavy cream", category: Category.DAIRY },
  { name: "Ketchup", category: Category.CONDIMENT },
  { name: "Mustard", category: Category.CONDIMENT },
  { name: "Mayo", category: Category.CONDIMENT },
  { name: "Hot sauce", category: Category.CONDIMENT },
  { name: "Soy sauce", category: Category.CONDIMENT },
  { name: "Salsa", category: Category.CONDIMENT },
  { name: "Ranch", category: Category.CONDIMENT },
  { name: "BBQ sauce", category: Category.CONDIMENT },
  { name: "Honey", category: Category.CONDIMENT },
  { name: "Jam", category: Category.CONDIMENT },
  { name: "Maple syrup", category: Category.CONDIMENT },
  { name: "Orange juice", category: Category.DRINK },
  { name: "Apple juice", category: Category.DRINK },
  { name: "Oat milk", category: Category.DRINK },
  { name: "Soda", category: Category.DRINK },
  { name: "Sparkling water", category: Category.DRINK },
  { name: "Energy drinks", category: Category.DRINK },
  { name: "Iced coffee", category: Category.DRINK },
  { name: "Chips", category: Category.SNACK },
  { name: "Popcorn", category: Category.SNACK },
  { name: "Granola bars", category: Category.SNACK },
  { name: "Cookies", category: Category.SNACK },
  { name: "Chocolate", category: Category.SNACK },
  { name: "Pretzels", category: Category.SNACK },
  { name: "Hummus", category: Category.SNACK },
  { name: "Leftover pizza", category: Category.LEFTOVER },
  { name: "Takeout", category: Category.LEFTOVER },
  { name: "Soup", category: Category.LEFTOVER },
];

/**
 * suggest is up to five common ingredients for what the player has typed: names that start with
 * it first, then names with a later word that does. Anything already on the list is left out.
 */
export function suggest(query: string, taken: readonly Item[]): Common[] {
  const typed = query.trim().toLowerCase();
  if (typed === "") {
    return [];
  }
  const have = new Set(taken.map((item) => item.name.toLowerCase()));
  const open = COMMON.filter((common) => !have.has(common.name.toLowerCase()));
  const first = open.filter((common) => common.name.toLowerCase().startsWith(typed));
  const later = open.filter(
    (common) =>
      !common.name.toLowerCase().startsWith(typed) &&
      common.name
        .toLowerCase()
        .split(" ")
        .some((word) => word.startsWith(typed)),
  );
  return [...first, ...later].slice(0, MAX_SUGGESTIONS);
}

/**
 * withIngredient is the list with `name` added last: under its own category and spelling when it
 * is a common ingredient, and under Other as typed when it isn't. A name the list already has, or
 * an empty one, leaves the list as it was.
 */
export function withIngredient(list: readonly Item[], name: string): readonly Item[] {
  const typed = name.trim();
  const same = (other: string) => other.toLowerCase() === typed.toLowerCase();
  if (typed === "" || list.some((item) => same(item.name))) {
    return list;
  }
  const common = COMMON.find((candidate) => same(candidate.name));
  return [
    ...list,
    { name: common?.name ?? typed, category: common?.category ?? Category.OTHER, quantity: "" },
  ];
}

/** withoutIngredient is the list without the ingredient at `index`. */
export function withoutIngredient(list: readonly Item[], index: number): readonly Item[] {
  return list.filter((_, i) => i !== index);
}

// One ingredient's field, as the api's buildEdit names it: ingredients[2].name.
const ITEM_FIELD = /^ingredients\[(\d+)\]\.\w+$/;

/** A refused edit's messages: one for the whole list, and one per ingredient by its place. */
export interface ListErrors {
  list: string | undefined;
  items: Map<number, string>;
}

/**
 * listErrors sorts a refused edit's violations by where they belong: the whole list
 * ("ingredients"), or one ingredient by its index ("ingredients[2].name"). An ingredient with two
 * problems shows the first.
 */
export function listErrors(invalid: Record<string, string>): ListErrors {
  const items = new Map<number, string>();
  for (const [field, message] of Object.entries(invalid)) {
    const index = ITEM_FIELD.exec(field)?.[1];
    if (index !== undefined && !items.has(Number(index))) {
      items.set(Number(index), message);
    }
  }
  return { list: invalid.ingredients, items };
}

/**
 * refusalOf is a refused save's error for as long as the draft is the list that was sent, and null
 * once it changed: the refusal names ingredients by their place in that list, and an edit moves
 * them. withIngredient and withoutIngredient hand back a new list for every change they make.
 */
export function refusalOf<E>(
  draft: readonly Item[],
  sent: readonly Item[] | null,
  error: E,
): E | null {
  return draft === sent ? error : null;
}
