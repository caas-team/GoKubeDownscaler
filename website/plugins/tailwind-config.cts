import { Plugin } from "@docusaurus/types";
import disableStyling from "./disable-styling.cts";
import disableTailwind from "./disable-tailwind.cts";

export function tailwindPlugin(): Plugin {
  return {
    name: "tailwind-plugin",
    configurePostCss(postcssOptions) {
      postcssOptions.plugins = [
        disableStyling,
        "@tailwindcss/postcss",
        disableTailwind,
      ];
      return postcssOptions;
    },
  };
}
