import { Plugin } from "unified";
import { visit } from "unist-util-visit";
import { Node, Literal } from "unist";
import { ParseFrontMatter } from "@docusaurus/types";

const references: Map<
  string,
  { urlPath: string; title: string; file: string }
> = new Map();

export const globalRefParseFrontMatter: ParseFrontMatter = async ({
  defaultParseFrontMatter,
  fileContent,
  filePath,
}) => {
  const result = await defaultParseFrontMatter({
    fileContent,
    filePath,
  });

  // file is not part of the docs or guides
  if (!filePath.includes("website/content/")) return result;

  // generate urlPath from filePath
  const urlPath = filePath
    .replace(/.*\/website\/content/g, "")
    .split("/")
    .map((part) => encodeURIComponent(part))
    .join("/");

  if (!result.frontMatter.globalReference) {
    throw new Error(`the file '${urlPath}' does not have a globalReference set`);
    return result;
  }

  const referenceId = result.frontMatter.globalReference as string;
  if (
    references.get(referenceId) &&
    references.get(referenceId).urlPath != urlPath
  ) {
    const errorMessage = `the globalReference '${referenceId}' is set in '${references.get(referenceId).urlPath}' and '${urlPath}'`;
    if (process.env.NODE_ENV === "production") {
      throw new Error(errorMessage);
    }
    console.error(
      `[ERROR] ${errorMessage}; if you moved/renamed this file you can ignore this warning`
    );
    return result;
  }

  // delete old reference for url path
  references.forEach((value, key) => {
    if (value.urlPath == urlPath) references.delete(key);
  });

  // set reference to urlPath
  references.set(referenceId, {
    urlPath,
    title: result.frontMatter.title as string,
    file: filePath,
  });

  return result;
};

export const docRefRemarkPlugin: Plugin = () => {
  return (tree: Node, file) => {
    const refPattern = /ref:([^#)]+)(?:#([^)]+))?/g;
    visit(tree, "link", (node: { url: string; title: string } & Literal) => {
      node.url = node.url.replace(
        refPattern,
        (match, referenceId, headerId) => {
          const reference = references.get(referenceId);

          // If the reference doesn't exist, log an error.
          if (!reference) {
            const errorMessage = `${file.path}:${node.position?.start.line}:${node.position?.start.column}: No reference found for '${referenceId}'`;
            if (process.env.NODE_ENV === "production") {
              throw new Error(errorMessage);
            }
            console.error(`[ERROR] ${errorMessage}`);
            return match;
          }

          if (file.path === reference.file) {
            const errorMessage = `${file.path}:${node.position?.start.line}:${node.position?.start.column}: Reference '${referenceId}' called in own file`;
            if (process.env.NODE_ENV === "production") {
              throw new Error(errorMessage);
            }
            console.error(`[ERROR] ${errorMessage}`);
            return match;
          }

          if (!node.title && reference.title) node.title = reference.title;

          // If there is a header ID, we append it to the URL
          return headerId
            ? `${reference.urlPath}#${headerId}`
            : reference.urlPath;
        }
      );
    });
  };
};
