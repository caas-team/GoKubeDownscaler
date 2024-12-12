import { MarkdownPreprocessor } from "@docusaurus/types/src/config";

const repoURL = "https://github.com/caas-team/GoKubeDownscaler";

export const repoRefPreprocessor: MarkdownPreprocessor = ({ fileContent }) => {
  // Regular expression to match repo reference. `repo`->link to repo, `repo:/path/to/file`, `repo:./path/to/file`, `repo:path/to/file`
  const refPattern = /\[([^\]]+)\]\(repo(?::([^)]+))?\)/g;

  // Replace repo references with corresponding repo URL
  return fileContent.replace(refPattern, (_, altText, repoPath) => {
    if (!repoPath) return `[${altText}](${repoURL})`;
    return `[${altText}](${repoURL}/tree/main/${repoPath})`;
  });
};
