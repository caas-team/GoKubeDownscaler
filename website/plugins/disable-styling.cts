import { type AcceptedPlugin, type PluginCreator } from "postcss";

// this will add an "useTailwind" css class which disables all css classes
// this is to be able to disable infimas/docusauruses globally applied element selectors
const disableStylingPlugin = (): AcceptedPlugin => {
  return {
    postcssPlugin: "disable-styling",
    Once(root) {
      root.walkRules((rule) => {
        if (rule.selector.includes(":not(.useTailwind)")) return;

        rule.selector = rule.selector
          .split(",")
          .map((selector) => {
            return `${selector}:not(.useTailwind)`;
          })
          .join(",");
      });
    },
  };
};

export default Object.assign(disableStylingPlugin, {
  postcss: true,
}) as PluginCreator<null>;
