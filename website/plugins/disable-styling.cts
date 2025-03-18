import { type AcceptedPlugin, type PluginCreator } from "postcss";

// this will add a "useTailwind" css class which disables all css classes
// this is needed to disable infimas/docusauruses globally applied element selectors
const disableStylingPlugin = (): AcceptedPlugin => {
  return {
    postcssPlugin: "disable-styling",
    Once(root) {
      root.walkRules((rule) => {
        if (
          rule.selector.includes(":not(.useTailwind)") ||
          rule.selector.startsWith(":") ||
          rule.selector.startsWith("*") //||
          //rule.selectors.some((selector) => selector.startsWith("."))
        )
          return;

        rule.selector = rule.selector
          .split(",")
          .map((selector) => {
            const parts = selector.split(":");
            parts.splice(1, 0, `not(.useTailwind):not(.useTailwind *)`);
            return parts.join(":");
          })
          .join(",");
      });
    },
  };
};

export default Object.assign(disableStylingPlugin, {
  postcss: true,
}) as PluginCreator<null>;
