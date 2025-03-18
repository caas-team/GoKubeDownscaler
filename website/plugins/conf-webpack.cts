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
        resolve: {
          fullySpecified: false,
          // manually resolve with extension
          alias: {
            "roughjs/bin/rough": "roughjs/bin/rough.js",
            "roughjs/bin/math": "roughjs/bin/math.js",
            "roughjs/bin/generator": "roughjs/bin/generator.js",
          },
        },
      };
    },
  };
}
