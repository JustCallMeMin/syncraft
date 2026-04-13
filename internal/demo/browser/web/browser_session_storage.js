const STORAGE_KEY = "syncraft:last-session-intent";

/**
 * loadSessionIntent reads the last browser session intent from localStorage.
 */
export function loadSessionIntent(storage = globalThis.localStorage) {
  if (!storage || typeof storage.getItem !== "function") {
    return null;
  }
  const raw = storage.getItem(STORAGE_KEY);
  if (typeof raw !== "string" || raw.trim() === "") {
    return null;
  }
  try {
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object") {
      return null;
    }
    if (typeof parsed.actorID !== "string" || parsed.actorID.trim() === "") {
      return null;
    }
    if (typeof parsed.documentID !== "string" || parsed.documentID.trim() === "") {
      return null;
    }
    return {
      actorID: parsed.actorID.trim(),
      documentID: parsed.documentID.trim(),
      retry: parsed.retry !== false,
    };
  } catch {
    return null;
  }
}

/**
 * saveSessionIntent persists the current browser session intent for refresh recovery.
 */
export function saveSessionIntent(intent, storage = globalThis.localStorage) {
  if (!storage || typeof storage.setItem !== "function") {
    return;
  }
  storage.setItem(STORAGE_KEY, JSON.stringify({
    actorID: intent.actorID,
    documentID: intent.documentID,
    retry: intent.retry !== false,
  }));
}
