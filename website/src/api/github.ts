import { adjustToLightHsl, hexToRGB, rgbToHSL } from "../util/color";

export const FetchGithubLabels = async ({
  org,
  repo,
}: FetchGithubLabelArgs): Promise<CachedGithubLabel[]> => {
  try {
    const response = await fetch(
      `https://api.github.com/repos/${org}/${repo}/labels`
    );
    if (!response.ok) {
      throw new Error(`got unexpected status '${response.status}' from github`);
    }
    const data = (await response.json()) as GithubLabel[];
    return data.map((label) => {
      const rgb = hexToRGB(label.color);
      const hsl = rgbToHSL(rgb);
      const lightHsl = adjustToLightHsl(hsl, rgb);

      return {
        name: label.name,
        description: label.description,
        rgb: rgb,
        hsl: hsl,
        lightHsl: lightHsl,
        url: `https://github.com/${org}/${repo}/labels/${label.name}`,
      };
    });
  } catch (error) {
    throw new Error(`Failed to fetch labels from github: ${error.message}`);
  }
};
