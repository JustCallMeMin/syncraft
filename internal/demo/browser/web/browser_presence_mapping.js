/**
 * runeIndexFromCodeUnitIndex converts a DOM textarea code-unit index into a rune index.
 */
export function runeIndexFromCodeUnitIndex(text, codeUnitIndex) {
  const source = typeof text === "string" ? text : "";
  const clamped = Math.max(0, Math.min(codeUnitIndex, source.length));
  return Array.from(source.slice(0, clamped)).length;
}

/**
 * codeUnitIndexFromRuneIndex converts a rune index back into a DOM textarea code-unit index.
 */
export function codeUnitIndexFromRuneIndex(text, runeIndex) {
  const source = typeof text === "string" ? text : "";
  if (runeIndex <= 0) {
    return 0;
  }
  let currentRuneIndex = 0;
  let currentCodeUnitIndex = 0;
  for (const rune of source) {
    if (currentRuneIndex >= runeIndex) {
      break;
    }
    currentCodeUnitIndex += rune.length;
    currentRuneIndex += 1;
  }
  return currentCodeUnitIndex;
}

/**
 * anchorForRuneIndex expresses one visible rune position against element identity plus offset.
 */
export function anchorForRuneIndex(runeIndex, visibleElements) {
  const elements = Array.isArray(visibleElements) ? visibleElements : [];
  const normalizedIndex = Math.max(0, runeIndex);
  if (elements.length === 0) {
    return { fallback_index: normalizedIndex };
  }

  let consumed = 0;
  for (const element of elements) {
    const runeCount = Array.from(element.value || "").length;
    if (normalizedIndex <= consumed + runeCount) {
      return {
        element_id: element.id,
        offset: Math.max(0, normalizedIndex - consumed),
        fallback_index: normalizedIndex,
      };
    }
    consumed += runeCount;
  }

  const last = elements[elements.length - 1];
  return {
    element_id: last.id,
    offset: Array.from(last.value || "").length,
    fallback_index: normalizedIndex,
  };
}

/**
 * runeIndexFromAnchor maps one element-anchored presence endpoint back into the current visible projection.
 */
export function runeIndexFromAnchor(anchor, visibleElements) {
  const elements = Array.isArray(visibleElements) ? visibleElements : [];
  const fallbackIndex = Number.isInteger(anchor?.fallback_index) ? Math.max(0, anchor.fallback_index) : 0;
  const elementID = typeof anchor?.element_id === "string" ? anchor.element_id : "";
  const offset = Number.isInteger(anchor?.offset) ? Math.max(0, anchor.offset) : 0;
  if (elementID === "") {
    return fallbackIndex;
  }

  let consumed = 0;
  for (const element of elements) {
    const runeCount = Array.from(element.value || "").length;
    if (element.id === elementID) {
      return consumed + Math.min(offset, runeCount);
    }
    consumed += runeCount;
  }
  return fallbackIndex;
}
