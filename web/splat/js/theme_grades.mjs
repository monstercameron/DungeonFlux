const THEMES = Object.freeze({
  neutral: { label: "Neutral", exposure: 0, contrast: 1, saturation: 1, shadow: [0, 0, 0], highlight: [0, 0, 0], strength: 0 },
  crypt: { label: "Crypt", exposure: -0.28, contrast: 1.1, saturation: 0.76, shadow: [0.06, 0.13, 0.1], highlight: [0.18, 0.1, 0.03], strength: 0.6 },
  flooded_hall: { label: "Flooded Hall", exposure: -0.25, contrast: 1.08, saturation: 0.8, shadow: [0.02, 0.1, 0.14], highlight: [0.08, 0.12, 0.16], strength: 0.7 },
  forest: { label: "Forest", exposure: -0.22, contrast: 1.08, saturation: 0.8, shadow: [0.05, 0.1, 0.06], highlight: [0.15, 0.11, 0.04], strength: 0.7 },
  harbor: { label: "Harbor", exposure: -0.3, contrast: 1.1, saturation: 0.76, shadow: [0.03, 0.07, 0.15], highlight: [0.16, 0.12, 0.04], strength: 0.7 },
  lava: { label: "Lava", exposure: -0.26, contrast: 1.12, saturation: 0.78, shadow: [0.12, 0.04, 0.03], highlight: [0.18, 0.07, 0.025], strength: 0.7 },
  library: { label: "Library", exposure: -0.24, contrast: 1.08, saturation: 0.78, shadow: [0.08, 0.05, 0.14], highlight: [0.14, 0.1, 0.18], strength: 0.7 },
  mountain: { label: "Mountain", exposure: -0.2, contrast: 1.06, saturation: 0.7, shadow: [0.04, 0.1, 0.14], highlight: [0.12, 0.15, 0.16], strength: 0.65 },
  swamp: { label: "Swamp", exposure: -0.35, contrast: 1.1, saturation: 0.72, shadow: [0.08, 0.12, 0.03], highlight: [0.16, 0.12, 0.03], strength: 0.7 },
  tavern: { label: "Tavern", exposure: -0.2, contrast: 1.08, saturation: 0.82, shadow: [0.03, 0.09, 0.08], highlight: [0.2, 0.13, 0.035], strength: 0.65 },
  throne: { label: "Throne", exposure: -0.24, contrast: 1.1, saturation: 0.76, shadow: [0.12, 0.04, 0.08], highlight: [0.18, 0.14, 0.06], strength: 0.7 },
});

const REFERENCE_METADATA = Object.freeze({
  crypt: { reference: "assets/concept/battlemap-crypt-sarcophagus-vault.jpg", shadow: "#15130c", highlight: "#765536", rationale: "malachite charcoal shadows with bronze vault light" },
  flooded_hall: { reference: "assets/concept/battlemap-flooded-hall-waterfall-temple.jpg", shadow: "#181c1b", highlight: "#b3744a", rationale: "slate cyan water shadows with amber torch light" },
  forest: { reference: "assets/concept/battlemap-forest-ruins-standing-stones.jpg", shadow: "#18150e", highlight: "#6d533f", rationale: "moss charcoal shadows with dusk mauve and aged gold" },
  harbor: { reference: "assets/concept/battlemap-harbor-docks-moonlit.jpg", shadow: "#111110", highlight: "#925e40", rationale: "ink navy shadows with lantern amber" },
  lava: { reference: "assets/concept/battlemap-lava-bridges-fortress-gate.jpg", shadow: "#1e0d04", highlight: "#f35f1d", rationale: "smoky umber shadows with copper fire" },
  library: { reference: "assets/concept/battlemap-library-orrery-reading-hall.jpg", shadow: "#201814", highlight: "#ab7755", rationale: "indigo slate shadows with lilac brass highlights" },
  mountain: { reference: "assets/concept/battlemap-mountain-ruins-cliff-bridges.jpg", shadow: "#1e2023", highlight: "#b77f62", rationale: "cool slate shadows with icy cyan and muted warmth" },
  swamp: { reference: "assets/concept/battlemap-swamp-boardwalk-shrine.jpg", shadow: "#1b150f", highlight: "#744e39", rationale: "olive teal shadows with ochre lamps" },
  tavern: { reference: "assets/concept/battlemap-tavern-flooded-common-room.jpg", shadow: "#140d07", highlight: "#824e2b", rationale: "teal umber shadows with honey gold" },
  throne: { reference: "assets/concept/battlemap-throne-room-marble-hall.jpg", shadow: "#26140c", highlight: "#9c6a4b", rationale: "burgundy slate shadows with polished gold" },
});

function finite(value, fallback) { return Number.isFinite(Number(value)) ? Number(value) : fallback; }
function clamp(value, min, max) { return Math.min(max, Math.max(min, value)); }
function tintFactors(values, amount) { const mean = values.reduce((sum, value) => sum + value, 0) / 3; return values.map((value) => 1 + (value - mean) * 1.5 * amount); }

export function getThemeGrade(theme = "neutral", strength) {
  const base = THEMES[theme] ?? THEMES.neutral;
  const amount = clamp(finite(strength, base.strength), 0, 1);
  return { theme: THEMES[theme] ? theme : "neutral", strength: amount, exposure: base.exposure * amount, contrast: 1 + (base.contrast - 1) * amount, saturation: 1 + (base.saturation - 1) * amount, shadow: base.shadow.map((v) => v * amount), highlight: base.highlight.map((v) => v * amount) };
}

export function applyGradeRGB(rgb, theme = "neutral", strength) {
  if (!Array.isArray(rgb) || rgb.length < 3) return [0, 0, 0];
  const grade = getThemeGrade(theme, strength);
  if (grade.theme === "neutral" || grade.strength === 0) return rgb.slice(0, 3).map(value => clamp(finite(value, 0), 0, 4));
  const exposed = rgb.slice(0, 3).map((value) => Math.max(0, finite(value, 0)) * (2 ** grade.exposure));
  const luma = exposed[0] * 0.2126 + exposed[1] * 0.7152 + exposed[2] * 0.0722;
  const shadow = tintFactors(grade.shadow, 1); const highlight = tintFactors(grade.highlight, 1); const mixLuma = clamp(luma, 0, 1);
  return exposed.map((value, index) => clamp((Math.max(0, luma + (value - luma) * grade.saturation) ** grade.contrast) * (shadow[index] * (1 - mixLuma) + highlight[index] * mixLuma), 0, 4));
}

export function gradeGLSL() {
  return `
    if (dfGradeEnabled > 0.5) {
      vec3 gradeRgb = max(color.rgb, vec3(0.0)) * exp2(dfGradeExposure);
      float gradeLuma = dot(gradeRgb, vec3(0.2126, 0.7152, 0.0722));
      gradeRgb = mix(vec3(gradeLuma), gradeRgb, dfGradeSaturation);
      gradeRgb = pow(max(gradeRgb, vec3(0.0)), vec3(dfGradeContrast));
      gradeRgb *= mix(dfGradeShadow, dfGradeHighlight, clamp(gradeLuma, 0.0, 1.0));
      color.rgb = clamp(gradeRgb, vec3(0.0), vec3(4.0));
    }
`;
}

export function gradeParameters(theme = "neutral", strength) {
  const grade = getThemeGrade(theme, strength);
  return { enabled: grade.theme !== "neutral" && grade.strength > 0 ? 1 : 0, exposure: grade.exposure, contrast: grade.contrast, saturation: grade.saturation, shadow: tintFactors(grade.shadow, 1), highlight: tintFactors(grade.highlight, 1) };
}

export { THEMES, REFERENCE_METADATA };
