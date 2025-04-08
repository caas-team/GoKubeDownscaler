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
import { confWebpack } from "./plugins/conf-webpack.cts";

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
    image: "img/social-preview.png",
    colorMode: {
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },
    navbar: {
      hideOnScroll: true,
      logo: {
        alt: "Kubedownscaler Logo",
        src: "img/kubedownscaler-name-dark.svg",
        srcDark: "img/kubedownscaler-name-light.svg",
      },
      title: "GoKubeDownscaler",
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
          type: "docSidebar",
          sidebarId: "ContributingSidebar",
          position: "left",
          label: "Contributing",
        },
        {
          href: "https://github.com/caas-team/GoKubeDownscaler",
          "aria-label": "GitHub",
          position: "right",
          title: "GoKubeDownscaler | Github",
          className: "navbar-icon icon-github",
        },
        {
          href: "https://inviter.co/kube-downscaler",
          "aria-label": "GitHub",
          position: "right",
          title: "kube-downscaler | Slack",
          className: "navbar-icon icon-slack",
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
      tagName: "script",
      attributes: {
        type: "application/ld+json",
      },
      innerHTML: JSON.stringify({
        "@context": "https://schema.org/",
        "@type": "SoftwareApplication",
        name: "GoKubeDownscaler",
        description:
          "GoKubeDownscaler is a Kubernetes autoscaler that lets you downscale your workloads to zero replicas during off-hours to save costs on cloud bill. It is lightweight and easy-to-use; works with EKS, GKE, AKS, and every other Kubernetes clusters.",
        applicationCategory: "Kubernetes Addon",
        operatingSystem: "Linux",
        url: "https://caas-team.github.io/GoKubeDownscaler/",
        logo: "https://github.com/caas-team/GoKubeDownscaler/blob/main/logo/kubedownscaler.svg",
        author: {
          "@type": "Organization",
          name: "CaaS Team",
          url: "https://github.com/caas-team",
        },
      }),
    },
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
  plugins: [tailwindPlugin, confWebpack],
  markdown: {
    parseFrontMatter: globalRefParseFrontMatter,
  },
};

export default config;
