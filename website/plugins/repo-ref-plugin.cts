import { Plugin } from "unified";
import { visit } from "unist-util-visit";
import { Node, Literal } from "unist";

const repoURL = "https://github.com/caas-team/GoKubeDownscaler";

export const repoRefRemarkPlugin: Plugin = () => {
  return (tree: Node, file) => {
    const refPattern = /repo(?::([^)]+))?/g;
    visit(tree, "link", (node: { url: string } & Literal) => {
      node.url = node.url.replace(refPattern, (match, repoPath) => {
        // if no path was provided, throw an error
        if (!repoPath) {
          const errorMessage = `${file.path}:${node.position?.start.line}:${node.position?.start.column}: No repository path specified`;
          if (process.env.NODE_ENV === "production") {
            throw new Error(errorMessage);
          }
          console.error(`[ERROR] ${errorMessage}`);
          return match;
        }
        return `${repoURL}/tree/main/${repoPath}`;
      });
    });
  };
};
