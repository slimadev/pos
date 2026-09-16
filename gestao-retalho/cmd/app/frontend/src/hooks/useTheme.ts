import { useEffect, useState } from "react";

export const THEMES = ["admin", "bottlestore", "ferragem", "mercearia", "farmacia"] as const;
export type Theme = (typeof THEMES)[number];

/**
 * Scaffolding-only theme switcher. From Fase 2 onward the active theme comes
 * from ConfigService (config.json's `tema`, SDD secção 9.3/9.6) instead of
 * being picked by hand here.
 */
export function useTheme(initial: Theme = "admin") {
  const [theme, setTheme] = useState<Theme>(initial);

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
  }, [theme]);

  return { theme, setTheme };
}
