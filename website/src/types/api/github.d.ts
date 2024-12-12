interface GithubLabel {
  name: string;
  description: string;
  color: string;
}

interface CachedGithubLabel {
  name: string;
  description: string;
  rgb: RgbColor;
  hsl: HslColor;
  lightHsl: HslColor;
  url: string;
}

interface FetchGithubLabelArgs {
  org: string;
  repo: string;
}
