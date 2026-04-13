/**
 * computeTextDiff resolves a single contiguous text diff using rune-aware indexes.
 */
export function computeTextDiff(previous, next) {
  const previousRunes = toRunes(previous);
  const nextRunes = toRunes(next);

  if (previousRunes.length === nextRunes.length && previousRunes.every((rune, index) => rune === nextRunes[index])) {
    return null;
  }

  let start = 0;
  while (
    start < previousRunes.length &&
    start < nextRunes.length &&
    previousRunes[start] === nextRunes[start]
  ) {
    start += 1;
  }

  let previousEnd = previousRunes.length;
  let nextEnd = nextRunes.length;
  while (
    previousEnd > start &&
    nextEnd > start &&
    previousRunes[previousEnd - 1] === nextRunes[nextEnd - 1]
  ) {
    previousEnd -= 1;
    nextEnd -= 1;
  }

  return {
    start,
    removed: previousRunes.slice(start, previousEnd).join(""),
    inserted: nextRunes.slice(start, nextEnd).join(""),
  };
}

/**
 * toRunes converts text into a rune-aware array for browser diffing.
 */
function toRunes(value) {
  if (typeof value !== "string") {
    return [];
  }
  return Array.from(value);
}
