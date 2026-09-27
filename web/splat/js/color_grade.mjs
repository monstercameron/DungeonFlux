import { THEMES, getThemeGrade, gradeGLSL, gradeParameters } from "./theme_grades.mjs";
import { setSplatColorGrade } from "./gray_skybox.mjs";

/** Applies a validated grade to a GSplat entity while preserving its sky predicate. */
export function applyColorGrade(entity, config, enabled = true) {
  if (!enabled || !config || config.enabled === false) return setSplatColorGrade(entity, "");
  const theme = typeof config.theme === "string" && Object.hasOwn(THEMES, config.theme) ? config.theme : "neutral";
  const grade = getThemeGrade(theme, config.strength);
  return setSplatColorGrade(entity, gradeGLSL(grade.theme, grade.strength), gradeParameters(grade.theme, grade.strength));
}

export function isKnownTheme(theme) { return typeof theme === "string" && Object.hasOwn(THEMES, theme); }
