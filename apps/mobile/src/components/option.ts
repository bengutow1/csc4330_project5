/** One choice in a pill toggle, segmented control or row of underline tabs. */
export interface Option<T extends string> {
  value: T;
  label: string;
}
