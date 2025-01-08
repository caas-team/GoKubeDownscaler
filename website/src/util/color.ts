export function rgbToHSL(rgb: RgbColor): HslColor {
  rgb = { r: rgb.r / 255, g: rgb.g / 255, b: rgb.b / 255 };

  const max = Math.max(rgb.r, rgb.g, rgb.b);
  const min = Math.min(rgb.r, rgb.g, rgb.b);
  const l = (max + min) / 2;

  let s = 0;
  if (max !== min) {
    s = l > 0.5 ? (max - min) / (2 - max - min) : (max - min) / (max + min);
  }

  let h = 0;
  if (max === rgb.r) {
    h = (rgb.g - rgb.b) / (max - min);
  } else if (max === rgb.g) {
    h = 2 + (rgb.b - rgb.r) / (max - min);
  } else if (max === rgb.b) {
    h = 4 + (rgb.r - rgb.g) / (max - min);
  }
  h = Math.round(h * 60);

  if (h < 0) h += 360;

  s = Math.round(s * 100);
  const lightness = Math.round(l * 100);

  return { h, s, l: lightness };
}

export function hexToRGB(hex: string): RgbColor {
  hex = hex.replace(/^#/, "");

  const r = parseInt(hex.slice(0, 2), 16);
  const g = parseInt(hex.slice(2, 4), 16);
  const b = parseInt(hex.slice(4, 6), 16);
  return { r, g, b };
}

export function adjustToLightHsl(hsl: HslColor, rgb: RgbColor): HslColor {
  const lightnessThreshold = 60 / 100;
  const percievedLightness = (rgb.r * 0.2126 + rgb.g * 0.7152 + rgb.b * 0.0722) / 255; // prettier-ignore
  const lightnessSwitch = Math.max(0, Math.min((1/(lightnessThreshold - percievedLightness)), 1)); // prettier-ignore
  const lightenBy = (lightnessThreshold - percievedLightness) * 100 * lightnessSwitch; // prettier-ignore
  return { h: hsl.h, s: hsl.s, l: hsl.l + lightenBy };
}
