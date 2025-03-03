import tsParser from "@typescript-eslint/parser";
import markdown from "eslint-plugin-markdown";
import path from "node:path";
import { includeIgnoreFile } from "@eslint/compat";
import { fileURLToPath } from "node:url";
import js from "@eslint/js";
import { FlatCompat } from "@eslint/eslintrc";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const compat = new FlatCompat({
  baseDirectory: __dirname,
  recommendedConfig: js.configs.recommended,
  allConfig: js.configs.all,
});

const gitignorePath = path.resolve(__dirname, "../.gitignore");

export default [
  {
    plugins: {
      markdown,
    },
  },
  ...compat.extends(
    "eslint:recommended",
    "plugin:@docusaurus/recommended",
    "plugin:@typescript-eslint/recommended"
  ),
  includeIgnoreFile(gitignorePath),
  {
    languageOptions: {
      parser: tsParser,
    },
  },
  {
    files: ["**/*.mdx"],
    processor: "markdown/markdown",
  },
];
