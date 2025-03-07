import { Plugin } from "@docusaurus/types";
import webpack from "webpack";

export function confWebpack(): Plugin {
  return {
    name: "configure-webpack",
    configureWebpack() {
      return {
        plugins: [
          new webpack.DefinePlugin({
            "process.env.IS_PREACT": JSON.stringify("false"),
          }),
        ],
        module: {
          rules: [
            {
              test: /\.excalidraw$/,
              use: "json-loader",
            },
          ],
        },
      };
    },
  };
}
