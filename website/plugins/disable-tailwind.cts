import { type AcceptedPlugin, type PluginCreator } from "postcss";

// this will add a "useTailwind" css class which disables tailwinds global css classes unless the "useTailwind" class is set
// this is needed to allow infima/docusaurus to do its thing without getting influenced by tailwind
const disableTailwindPlugin = (): AcceptedPlugin => {
  return {
    postcssPlugin: "disable-tailwind",
    Once(root) {
      root.walkRules((rule) => {
        if (
          rule.selector.includes(":not(.useTailwind)") ||
          rule.selector.startsWith(":")
        )
          return;

        rule.selector = rule.selector
          .split(",")
          .map((selector) => {
            return `.useTailwind ${selector}`;
          })
          .join(",");
      });
    },
  };
};

export default Object.assign(disableTailwindPlugin, {
  postcss: true,
}) as PluginCreator<null>;
