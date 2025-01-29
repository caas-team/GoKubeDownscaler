/* eslint-disable @typescript-eslint/no-require-imports */
import { Plugin } from "@docusaurus/types";

export function tailwindPlugin(
): Plugin {
  return {
    name: "tailwind-plugin",
    configurePostCss(postcssOptions) {
      postcssOptions.plugins = [
        require("postcss-import"),
        require("tailwindcss"),
        require("autoprefixer"),
      ];
      return postcssOptions;
    },
  };
}
