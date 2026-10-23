/**
 * foodSlug is the file name a food's photo is kept under: lowercase words joined by dashes, so
 * "Lemon Water (Fancy)" is lemon-water-fancy. A single egg shares the carton's photo.
 */
export function foodSlug(name: string): string {
  const slug = name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
  return ALIASES[slug] ?? slug;
}

const ALIASES: Record<string, string> = { egg: "eggs" };
