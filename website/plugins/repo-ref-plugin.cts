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
          console.error(
            `"%s:%d:%d": No repository path specified`,
            file.path,
            node.position.start.line,
            node.position.start.column
          );
          return match;
        }
        return `${repoURL}/tree/main/${repoPath}`;
      });
    });
  };
};
