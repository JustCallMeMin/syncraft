const palette = [
  "#146c94",
  "#2d7c67",
  "#9c6644",
  "#7c3aed",
  "#d9485f",
  "#b7791f",
  "#2563eb",
  "#047857",
];

/**
 * colorTokenForActor returns a stable collaborator color for one actor instance id.
 */
export function colorTokenForActor(actorID) {
  const normalized = typeof actorID === "string" ? actorID.trim() : "";
  if (normalized === "") {
    return palette[0];
  }
  let hash = 0;
  for (const char of normalized) {
    hash = (hash * 31 + char.codePointAt(0)) >>> 0;
  }
  return palette[hash % palette.length];
}
