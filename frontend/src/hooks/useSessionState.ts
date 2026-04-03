import { useState, useCallback } from "react";

/**
 * A drop-in replacement for useState that persists the value
 * in sessionStorage. The value survives page refreshes within
 * the same browser tab but is cleared when the tab is closed.
 *
 * @param key - The sessionStorage key to store under.
 * @param defaultValue - Fallback when no stored value exists.
 * @returns [value, setValue] with the same API as useState.
 */
function useSessionState<T>(
  key: string,
  defaultValue: T
): [T, (value: T | ((prev: T) => T)) => void] {
  const [state, setState] = useState<T>(() => {
    try {
      const stored = sessionStorage.getItem(key);
      if (stored !== null) {
        return JSON.parse(stored) as T;
      }
    } catch {
      // Invalid JSON or sessionStorage unavailable
    }
    return defaultValue;
  });

  const setValue = useCallback(
    (value: T | ((prev: T) => T)) => {
      setState((prev) => {
        const next =
          value instanceof Function ? value(prev) : value;
        try {
          sessionStorage.setItem(key, JSON.stringify(next));
        } catch {
          // sessionStorage full or unavailable
        }
        return next;
      });
    },
    [key]
  );

  return [state, setValue];
}

export default useSessionState;
