import { Plugin } from "unified";
import { visit } from "unist-util-visit";
import { Node, Literal } from "unist";
import { ParseFrontMatter } from "@docusaurus/types";

const references: Map<string, { urlPath: string; title: string }> = new Map();

export const globalRefParseFrontMatter: ParseFrontMatter = async ({
  defaultParseFrontMatter,
  fileContent,
  filePath,
}) => {
  const result = await defaultParseFrontMatter({
    fileContent,
    filePath,
  });

  if (!result.frontMatter.globalReference) return result;

  // generate urlPath from filePath
  const urlPath = filePath
    .replace(/.*\/website\/content/g, "")
    .split("/")
    .map((part) => encodeURIComponent(part))
    .join("/");

  const referenceId = result.frontMatter.globalReference as string;
  if (
    references.get(referenceId) &&
    references.get(referenceId).urlPath != urlPath
  )
    console.warn(
      "the globalReference '%s' is set in '%s' and '%s'. if you moved/renamed this file you can ignore this warning.",
      referenceId,
      references.get(referenceId).urlPath,
      urlPath
    );

  // delete old reference for url path
  references.forEach((value, key) => {
    if (value.urlPath == urlPath) references.delete(key);
  });

  // set reference to urlPath
  references.set(referenceId, {
    urlPath,
    title: result.frontMatter.title as string,
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
            console.error(
              `No reference found for '%s'. File: '%s', Node: '%o'`,
              referenceId,
              file.path,
              node
            );
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
