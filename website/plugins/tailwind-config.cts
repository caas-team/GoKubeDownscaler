import { LoadContext, Plugin, PluginOptions } from "@docusaurus/types";

export function tailwindPlugin(
  context: LoadContext,
  options: PluginOptions
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
