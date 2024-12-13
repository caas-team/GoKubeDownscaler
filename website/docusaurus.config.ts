import { themes as prismThemes } from "prism-react-renderer";
import type { Config } from "@docusaurus/types";
import type * as Preset from "@docusaurus/preset-classic";
import { tailwindPlugin } from "./plugins/tailwind-config.cts";
import { svgoConfigPlugin } from "./plugins/svgo-config.cts";
import {
  docRefRemarkPlugin,
  globalRefParseFrontMatter,
} from "./plugins/global-ref-plugin.cts";
import { repoRefRemarkPlugin } from "./plugins/repo-ref-plugin.cts";

const config: Config = {
  title: "GoKubeDownscaler",
  tagline: "A horizontal autoscaler for Kubernetes workloads",
  favicon: "img/CaaS-Logo.svg",

  url: "https://caas-team.github.io",

  baseUrl: "/GoKubeDownscaler",

  organizationName: "caas-team",
  projectName: "GoKubeDownscaler",

  trailingSlash: false,

  onBrokenLinks: "throw",
  onBrokenMarkdownLinks: "warn",

  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },

  presets: [
    [
      "classic",
      {
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
      title: "GoKubeDownscaler",
      logo: {
        alt: "CaaS Logo",
        src: "img/CaaS-Logo.svg",
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
        { to: "/about", label: "About", position: "left" },
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
      links: [
        {
          title: "Content",
          items: [
            {
              label: "Documentation",
              to: "/docs",
            },
            {
              label: "Guides",
              to: "/guides",
            },
          ],
        },
        {
          title: "Community",
          items: [
            {
              label: "Slack",
              href: "https://communityinviter.com/apps/kube-downscaler/kube-downscaler",
            },
            {
              label: "GitHub",
              href: "https://github.com/caas-team/GoKubeDownscaler",
            },
          ],
        },
        {
          title: "More",
          items: [
            {
              label: "About",
              to: "/about",
            },
          ],
        },
      ],
      copyright: `Copyright © GoKubeDownscaler Authors ${new Date().getFullYear()}`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ["mdx", "bash"],
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
  plugins: [svgoConfigPlugin, tailwindPlugin],
  markdown: {
    parseFrontMatter: globalRefParseFrontMatter,
  },
};

export default config;
