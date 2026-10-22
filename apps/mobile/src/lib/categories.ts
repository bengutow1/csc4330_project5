import { Category } from "@fries-in-the-bag/api";
import type { IconName } from "../components/icon-button.tsx";
import type { Item } from "./ingredients.ts";

/** How a category shows: the heading over its ingredients, and the icon on each of their tiles. */
export interface CategoryLook {
  label: string;
  icon: IconName;
}

/** One category's ingredients, each with its place in the whole list. */
export interface Group {
  category: Category;
  items: { item: Item; index: number }[];
}

// A category added to the wire fails to compile here until it has a look.
const LOOKS: Record<Exclude<Category, Category.UNSPECIFIED>, CategoryLook> = {
  [Category.PROTEIN]: { label: "Protein", icon: "egg-outline" },
  [Category.VEGETABLE]: { label: "Vegetables", icon: "leaf-outline" },
  [Category.FRUIT]: { label: "Fruit", icon: "nutrition-outline" },
  [Category.GRAIN]: { label: "Grains", icon: "basket-outline" },
  [Category.DAIRY]: { label: "Dairy", icon: "water-outline" },
  [Category.CONDIMENT]: { label: "Condiments", icon: "flask-outline" },
  [Category.DRINK]: { label: "Drinks", icon: "cafe-outline" },
  [Category.SNACK]: { label: "Snacks", icon: "ice-cream-outline" },
  [Category.LEFTOVER]: { label: "Leftovers", icon: "pizza-outline" },
  [Category.OTHER]: { label: "Other", icon: "cube-outline" },
};

// The category a group files an ingredient under: its own, or Other when the app doesn't know it.
// Proto enums are open, so a newer server can send one.
function known(category: Category): Category {
  const looks: Partial<Record<Category, CategoryLook>> = LOOKS;
  return looks[category] === undefined ? Category.OTHER : category;
}

/** categoryLook is how a category shows. One the app doesn't know shows as Other. */
export function categoryLook(category: Category): CategoryLook {
  const looks: Partial<Record<Category, CategoryLook>> = LOOKS;
  return looks[category] ?? LOOKS[Category.OTHER];
}

/**
 * groupByCategory sorts a list into its categories for the verdict's sections and the edit sheet.
 * Groups come in the order their category first appears and each keeps the server's order, so
 * nothing is re-ranked. Each ingredient keeps its index in the list, which is how an edit or an
 * error names it.
 */
export function groupByCategory(list: readonly Item[]): Group[] {
  const groups: Group[] = [];
  list.forEach((item, index) => {
    const category = known(item.category);
    const group = groups.find((g) => g.category === category);
    if (group === undefined) {
      groups.push({ category, items: [{ item, index }] });
    } else {
      group.items.push({ item, index });
    }
  });
  return groups;
}
