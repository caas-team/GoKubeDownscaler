import { Plugin } from "@docusaurus/types";
import * as PostCssImport from "postcss-import";
import * as TailwindCss from "tailwindcss";
import * as Autoprefixer from "autoprefixer";

export function tailwindPlugin(
): Plugin {
  return {
    name: "tailwind-plugin",
    configurePostCss(postcssOptions) {
      postcssOptions.plugins = [
        PostCssImport,
        TailwindCss,
        Autoprefixer,
      ];
      return postcssOptions;
    },
  };
}
