import { themes as prismThemes } from "prism-react-renderer";
import type { Config } from "@docusaurus/types";
import type * as Preset from "@docusaurus/preset-classic";

const config: Config = {
  title: "GoKubeDownscaler",
  tagline: "A vertical autoscaler for Kubernetes workloads",
  favicon: "img/CaaS-Logo.svg",

  url: "https://caas-team.github.io",

  baseUrl: "/",

  organizationName: "caas-team",
  projectName: "GoKubeDownscaler",

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
          path: "documenation",
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
      content:
        '⭐️ If you like GoKubeDownscaler, give it a star on <a target="_blank" rel="noopener noreferrer" href="https://github.com/caas-team/GoKubeDownscaler">GitHub</a>! ⭐️',
    },
    footer: {
      style: "dark",
      links: [
        {
          title: "Documentation",
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
      copyright: `Copyright © ${new Date().getFullYear()} Deutsche Telekom AG`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
