import { themes as prismThemes } from "prism-react-renderer";
import type { Config } from "@docusaurus/types";
import type * as Preset from "@docusaurus/preset-classic";
import { tailwindPlugin } from "./plugins/tailwind-config.cts";
import {
  docRefRemarkPlugin,
  globalRefParseFrontMatter,
} from "./plugins/global-ref-plugin.cts";
import { repoRefRemarkPlugin } from "./plugins/repo-ref-plugin.cts";
import { PluginOptions } from "@easyops-cn/docusaurus-search-local";
import { PluginConfig } from "svgo/lib/svgo";
import path from "path";

const config: Config = {
  title: "GoKubeDownscaler",
  tagline: "A horizontal autoscaler for Kubernetes workloads",
  favicon: "img/kubedownscaler.svg",

  url: "https://caas-team.github.io",

  baseUrl: "/GoKubeDownscaler",

  organizationName: "caas-team",
  projectName: "GoKubeDownscaler",

  trailingSlash: false,

  onBrokenLinks: "throw",
  onBrokenMarkdownLinks: "throw",
  onBrokenAnchors: "throw",

  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },

  presets: [
    [
      "classic",
      {
        svgr: {
          svgrConfig: {
            svgoConfig: {
              plugins: [
                "preset-default", // extend default config
                "removeDimensions", // automatically switch from width and height to viewbox
                {
                  // prefix ids and class names with the filename, to prevent duplicate ids from interfering with eachother
                  name: "prefixIds",
                  params: {
                    delim: "_",
                    prefix: (_, file) => {
                      return path.basename(file?.path ?? "").split(".")[0];
                    },
                    prefixIds: true,
                    prefixClassNames: true,
                  },
                },
              ] satisfies PluginConfig[],
            },
          },
        },
        docs: {
          sidebarPath: "./sidebars.ts",
          routeBasePath: "/",
          path: "content",
          beforeDefaultRemarkPlugins: [docRefRemarkPlugin, repoRefRemarkPlugin],
          editUrl:
            "https://github.com/caas-team/GoKubeDownscaler/edit/main/website",
        },
        theme: {
          customCss: "./src/css/custom.css",
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    colorMode: {
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },
    navbar: {
      logo: {
        alt: "CaaS Logo",
        src: "img/kubedownscaler-name-dark.svg",
        srcDark: "img/kubedownscaler-name-light.svg",
      },
      items: [
        {
          type: "docSidebar",
          sidebarId: "DocsSidebar",
          position: "left",
          label: "Documentation",
        },
        {
          type: "docSidebar",
          sidebarId: "GuidesSidebar",
          position: "left",
          label: "Guides",
        },
        {
          href: "https://github.com/caas-team/GoKubeDownscaler",
          label: "GitHub",
          position: "right",
        },
        {
          href: "https://communityinviter.com/apps/kube-downscaler/kube-downscaler",
          label: "Slack",
          position: "right",
        },
      ],
    },
    announcementBar: {
      id: "star_downscaler",
      content:
        '⭐️ If you like GoKubeDownscaler, give it a star on <a target="_blank" rel="noopener noreferrer" href="https://github.com/caas-team/GoKubeDownscaler">GitHub</a>! ⭐️',
    },
    footer: {
      style: "dark",
      copyright: `Copyright © GoKubeDownscaler Authors ${new Date().getFullYear()}`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ["mdx", "bash"],
      magicComments: [
        {
          className: "theme-code-block-highlighted-line",
          line: "highlight-next-line",
          block: { start: "highlight-start", end: "highlight-end" },
        },
      ],
    },
  } satisfies Preset.ThemeConfig,
  headTags: [
    {
      tagName: "link",
      attributes: {
        rel: "manifest",
        href: "/GoKubeDownscaler/manifest.json",
      },
    },
  ],
  themes: [
    [
      require.resolve("@easyops-cn/docusaurus-search-local"),
      {
        hashed: true,
        indexBlog: false,
        docsRouteBasePath: ["/docs", "/guides"],
        docsDir: "content",
        searchBarShortcutHint: false,
      } as Partial<PluginOptions>,
    ],
  ],
  plugins: [tailwindPlugin],
  markdown: {
    parseFrontMatter: globalRefParseFrontMatter,
  },
};

export default config;
