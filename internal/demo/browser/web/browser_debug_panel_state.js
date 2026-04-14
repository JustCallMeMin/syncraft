/**
 * createDebugPanelState manages the collapsed or expanded state of the in-app debug panel shell.
 */
export function createDebugPanelState(initialExpanded = false) {
  let expanded = Boolean(initialExpanded);

  return {
    /**
     * isExpanded returns the current expansion state for the debug panel shell.
     */
    isExpanded() {
      return expanded;
    },

    /**
     * setExpanded forces the debug panel shell into one explicit expansion state.
     */
    setExpanded(nextExpanded) {
      expanded = Boolean(nextExpanded);
      return expanded;
    },

    /**
     * toggle flips the current debug panel shell state and returns the new state.
     */
    toggle() {
      expanded = !expanded;
      return expanded;
    },
  };
}
