import { Plugin } from "@docusaurus/types";
import disableStyling from "./disable-styling.cts";

export function tailwindPlugin(): Plugin {
  return {
    name: "tailwind-plugin",
    configurePostCss(postcssOptions) {
      postcssOptions.plugins = [disableStyling, "@tailwindcss/postcss"];
      return postcssOptions;
    },
  };
}
