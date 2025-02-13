import { Plugin } from "unified";
import { visit } from "unist-util-visit";
import { Node, Literal } from "unist";

const repoURL = "https://github.com/caas-team/GoKubeDownscaler";

export const repoRefRemarkPlugin: Plugin = () => {
  return (tree: Node) => {
    const refPattern = /repo(?::([^)]+))?/g;
    visit(tree, "link", (node: { url: string } & Literal) => {
      node.url = node.url.replace(refPattern, (_, repoPath) => {
        if (!repoPath) return repoURL;
        return `${repoURL}/tree/main/${repoPath}`;
      });
    });
  };
};
