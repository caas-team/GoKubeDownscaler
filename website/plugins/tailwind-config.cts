import { Plugin } from "@docusaurus/types";

export function tailwindPlugin(): Plugin {
  return {
    name: "tailwind-plugin",
    configurePostCss(postcssOptions) {
      postcssOptions.plugins = ["@tailwindcss/postcss"];
      return postcssOptions;
    },
  };
}
