/** The campus board, which every player is on: the scope with no crew. */
export const CAMPUS = 0n;

/** CrewLike is what these helpers read from a crew, as the api sends it. */
export interface CrewLike {
  crewId: bigint;
  memberCount: number;
}

/**
 * scopeOf is the board scope to show: the crew the player picked while they are still in it, and
 * the campus once they are not (they left, or the list has not loaded yet). Derived each render,
 * so leaving a crew can never strand the board on one the player cannot see.
 */
export function scopeOf(crews: readonly CrewLike[] | undefined, picked: bigint): bigint {
  if (picked === CAMPUS) {
    return CAMPUS;
  }
  return crews?.some((crew) => crew.crewId === picked) ? picked : CAMPUS;
}

/** memberLine is the line under a crew's name: how many are in it. */
export function memberLine(crew: CrewLike): string {
  const n = crew.memberCount;
  return `${n} member${n === 1 ? "" : "s"}`;
}
