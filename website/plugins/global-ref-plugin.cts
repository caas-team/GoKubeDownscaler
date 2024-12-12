import { ParseFrontMatter } from "@docusaurus/types";
import { MarkdownPreprocessor } from "@docusaurus/types/src/config";

const references: Map<string, string> = new Map();

export const globalRefParseFrontMatter: ParseFrontMatter = async ({
  defaultParseFrontMatter,
  fileContent,
  filePath,
}) => {
  const result = await defaultParseFrontMatter({
    fileContent,
    filePath,
  });

  // generate urlPath from filePath
  const urlPath = filePath
    .replace(/.*\/website\/content/g, "")
    .split("/")
    .map((part) => encodeURIComponent(part))
    .join("/");

  // delete old reference
  references.forEach((value, key) => {
    if (value == urlPath) references.delete(key);
  });

  // set reference to urlPath
  if (result.frontMatter.globalReference)
    references.set(result.frontMatter.globalReference as string, urlPath);

  return result;
};

export const globalRefPreprocessor: MarkdownPreprocessor = ({
  fileContent,
  filePath,
}) => {
  // Regular expression to match both patterns: ref:example-id and ref:example-id#header-id
  const refPattern = /\[([^\]]+)\]\(ref:([^#)]+)(?:#([^)]+))?\)/g;

  // Replace references with corresponding URLs if they exist in `references`
  return fileContent.replace(
    refPattern,
    (match, altText, reference, headerId) => {
      const referenceUrl = references.get(reference);

      // If the reference exists in the references map
      if (referenceUrl) {
        // If there is a header ID, we append it to the URL
        const newUrl = headerId ? `${referenceUrl}#${headerId}` : referenceUrl;
        return `[${altText}](${newUrl})`;
      }

      // If the reference doesn't exist, error.
      console.error(
        "no reference found for '%s'. full match: '%s' in '%s'",
        reference,
        match,
        filePath
      );
      return `\\[${altText}\\]\\(ref:${reference}${
        headerId ? "#" + headerId : ""
      }\\)`;
    }
  );
};
