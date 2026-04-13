/**
 * loadSessionIntent reads the current tab's actor/document intent from the URL query string.
 */
export function loadSessionIntent(locationLike = globalThis.location) {
  const search = typeof locationLike?.search === "string" ? locationLike.search : "";
  const params = new URLSearchParams(search);
  const actorID = params.get("actor");
  const documentID = params.get("document");
  const retry = params.get("retry");

  if (typeof actorID !== "string" || actorID.trim() === "") {
    return null;
  }
  if (typeof documentID !== "string" || documentID.trim() === "") {
    return null;
  }

  return {
    actorID: actorID.trim(),
    documentID: documentID.trim(),
    retry: retry !== "false",
  };
}

/**
 * saveSessionIntent persists the current tab's actor/document intent into the URL query string.
 */
export function saveSessionIntent(intent, locationLike = globalThis.location, historyLike = globalThis.history) {
  if (!locationLike || typeof locationLike.href !== "string") {
    return;
  }
  if (!historyLike || typeof historyLike.replaceState !== "function") {
    return;
  }

  const url = new URL(locationLike.href);
  url.searchParams.set("actor", intent.actorID);
  url.searchParams.set("document", intent.documentID);
  if (intent.retry === false) {
    url.searchParams.set("retry", "false");
  } else {
    url.searchParams.delete("retry");
  }
  historyLike.replaceState(null, "", `${url.pathname}${url.search}${url.hash}`);
}
