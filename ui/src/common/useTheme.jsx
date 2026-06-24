import { createContext, useContext, useEffect, useState, useCallback } from "react";

const STORAGE_KEY = "inkstone-theme";
const MODES = ["light", "dark", "system"];

const ThemeContext = createContext(null);

function getStoredMode() {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    return MODES.includes(saved) ? saved : "system";
  } catch {
    return "system";
  }
}

function systemPrefersDark() {
  return (
    typeof window !== "undefined" &&
    window.matchMedia &&
    window.matchMedia("(prefers-color-scheme: dark)").matches
  );
}

function resolve(mode) {
  if (mode === "system") return systemPrefersDark() ? "dark" : "light";
  return mode;
}

function apply(resolved) {
  document.documentElement.setAttribute("data-bs-theme", resolved);
}

export function ThemeProvider({ children }) {
  const [mode, setModeState] = useState(getStoredMode);
  const [resolved, setResolved] = useState(() => resolve(getStoredMode()));

  // Apply on mount and whenever the chosen mode changes.
  useEffect(() => {
    const r = resolve(mode);
    setResolved(r);
    apply(r);
    try {
      localStorage.setItem(STORAGE_KEY, mode);
    } catch {
      /* ignore quota / privacy-mode errors */
    }
  }, [mode]);

  // Follow OS changes while in "system" mode.
  useEffect(() => {
    if (mode !== "system" || !window.matchMedia) return;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => {
      const r = systemPrefersDark() ? "dark" : "light";
      setResolved(r);
      apply(r);
    };
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, [mode]);

  const setMode = useCallback((m) => {
    if (MODES.includes(m)) setModeState(m);
  }, []);

  // Cycle light → dark → system → light (used by the navbar control).
  const cycleMode = useCallback(() => {
    setModeState((m) => MODES[(MODES.indexOf(m) + 1) % MODES.length]);
  }, []);

  return (
    <ThemeContext.Provider value={{ mode, resolved, setMode, cycleMode }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error("useTheme must be used within a ThemeProvider");
  return ctx;
}
