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
import {
  firstDocRedirectPlugin,
  Config as firstDocRedirectConfig,
} from "./plugins/first-doc-redirect.cts";

const config: Config = {
  title: "GoKubeDownscaler",
  tagline: "A Horizontal Autoscaler For Kubernetes Workloads",
  favicon: "img/kubedownscaler.svg",

  url: "https://kube-downscaler.io",

  baseUrl: "/",

  organizationName: "caas-team",
  projectName: "GoKubeDownscaler",

  trailingSlash: false,

  onBrokenLinks: "throw",
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
          showLastUpdateTime: true,
          versions: {
            current: { label: "Next 🚧" },
          },
        },
        blog: {
          blogTitle: "GoKubeDownscaler Blog",
          blogDescription: "The official blog of the GoKubeDownscaler",
          postsPerPage: "ALL",
          showReadingTime: true,
          editUrl:
            "https://github.com/caas-team/GoKubeDownscaler/edit/main/website",
          onInlineTags: "throw",
          onInlineAuthors: "throw",
          onUntruncatedBlogPosts: "throw",
          showLastUpdateTime: true,
          beforeDefaultRemarkPlugins: [repoRefRemarkPlugin],
        },
        theme: {
          customCss: "./src/css/custom.css",
        },
        sitemap: {
          changefreq: "weekly",
          priority: 0.8,
          ignorePatterns: ["/tags/**"],
          filename: "sitemap.xml",
        },
      } satisfies Preset.Options,
    ],
  ],

  // see https://github.com/facebook/docusaurus/issues/10556
  // this is necessary for tailwind since the old css minifier removes the layer from @media css rules
  // additionally this makes building faster. if we ever get issues from this we can manually just enable the new css minimizer
  future: { faster: true, v4: true },

  themeConfig: {
    image: "img/social-preview.png",
    colorMode: {
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },
    navbar: {
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
          to: "/adopters",
          label: "Adopters",
          position: "left",
        },
        {
          to: "blog",
          label: "Blog",
          position: "left",
        },
        {
          type: "docsVersionDropdown",
          position: "right",
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
          "aria-label": "Slack Community",
          position: "right",
          title: "kube-downscaler | Slack",
          className: "navbar-icon icon-slack",
        },
        {
          href: "https://kube-downscaler.io/blog/rss.xml",
          "aria-label": "RSS Feed",
          position: "right",
          title: "GoKubeDownscaler Blog | RSS Feed",
          className: "navbar-icon icon-rss",
        },
      ],
    },
    announcementBar: {
      id: "star_downscaler",
      content:
        '<span class="announcement-full">⭐️ If you like GoKubeDownscaler, give it a star on <a target="_blank" rel="noopener noreferrer" href="https://github.com/caas-team/GoKubeDownscaler">GitHub</a>! ⭐️</span><span class="announcement-short">⭐️ Give it a star on <a target="_blank" rel="noopener noreferrer" href="https://github.com/caas-team/GoKubeDownscaler">GitHub</a>! ⭐️</span>',
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
    mermaid: {
      theme: { light: "neutral", dark: "dark" },
    },
  } satisfies Preset.ThemeConfig,
  headTags: [
    // Global Open Graph tags (apply to every page)
    {
      tagName: "meta",
      attributes: {
        property: "og:type",
        content: "website",
      },
    },
    {
      tagName: "meta",
      attributes: {
        property: "og:site_name",
        content: "GoKubeDownscaler",
      },
    },
    // Default OG image dimensions (supplements themeConfig.image which injects og:image)
    {
      tagName: "meta",
      attributes: {
        property: "og:image:width",
        content: "1280",
      },
    },
    {
      tagName: "meta",
      attributes: {
        property: "og:image:height",
        content: "640",
      },
    },
    {
      tagName: "meta",
      attributes: {
        property: "og:image:alt",
        content: "GoKubeDownscaler — Kubernetes Scheduled Autoscaler",
      },
    },
    // Global Twitter Card type
    {
      tagName: "meta",
      attributes: {
        name: "twitter:card",
        content: "summary_large_image",
      },
    },
    // Preconnect for Google Fonts to improve LCP (non-render-blocking)
    {
      tagName: "link",
      attributes: {
        rel: "preconnect",
        href: "https://fonts.googleapis.com",
      },
    },
    {
      tagName: "link",
      attributes: {
        rel: "preconnect",
        href: "https://fonts.gstatic.com",
        crossorigin: "anonymous",
      },
    },
    {
      tagName: "link",
      attributes: {
        rel: "preload",
        as: "style",
        href: "https://fonts.googleapis.com/css2?family=Poppins:wght@700&display=swap",
        onload: "this.onload=null;this.rel='stylesheet'",
      },
    },
    // Preload hero SVG for faster LCP (Largest Contentful Paint)
    {
      tagName: "link",
      attributes: {
        rel: "preload",
        as: "image",
        href: "/img/kubedownscaler.svg",
        type: "image/svg+xml",
      },
    },
    {
      tagName: "noscript",
      innerHTML:
        '<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Poppins:wght@700&display=swap">',
      attributes: {},
    },
    // SoftwareApplication structured data (enriched)
    {
      tagName: "script",
      attributes: {
        type: "application/ld+json",
      },
      innerHTML: JSON.stringify({
        "@context": "https://schema.org/",
        "@type": "SoftwareApplication",
        "@id": "https://kube-downscaler.io/#software",
        name: "GoKubeDownscaler",
        alternateName: [
          "kube-downscaler",
          "go-kube-downscaler",
          "kubernetes downscaler",
        ],
        description:
          "GoKubeDownscaler is a horizontal autoscaler that scales Kubernetes workloads down during off-hours (nights, weekend, holidays) to reduce cloud costs",
        image: "https://kube-downscaler.io/img/social-preview.png",
        applicationCategory: "DeveloperApplication",
        applicationSubCategory: "Kubernetes Addon",
        operatingSystem: "Linux",
        url: "https://kube-downscaler.io/",
        downloadUrl:
          "https://github.com/caas-team/GoKubeDownscaler/releases/latest",
        releaseNotes: "https://github.com/caas-team/GoKubeDownscaler/releases",
        license: "https://opensource.org/licenses/Apache-2.0",
        offers: {
          "@type": "Offer",
          price: "0",
          priceCurrency: "USD",
        },
        sameAs: ["https://github.com/caas-team/GoKubeDownscaler"],
        keywords:
          "kubernetes, kube-downscaler, downscaler, cost optimization, scheduled scaling, cloud costs, kubernetes autoscaler",
        author: { "@id": "https://kube-downscaler.io/#organization" },
        maintainer: { "@id": "https://kube-downscaler.io/#organization" },
        publisher: { "@id": "https://kube-downscaler.io/#organization" },
        codeRepository: "https://github.com/caas-team/GoKubeDownscaler",
      }),
    },
    // WebSite schema with Sitelinks
    {
      tagName: "script",
      attributes: {
        type: "application/ld+json",
      },
      innerHTML: JSON.stringify({
        "@context": "https://schema.org",
        "@type": "WebSite",
        "@id": "https://kube-downscaler.io/#website",
        name: "GoKubeDownscaler",
        url: "https://kube-downscaler.io/",
        description:
          "GoKubeDownscaler: a Kubernetes horizontal autoscaler that reduces Kubernetes cloud costs by scaling workloads based on time schedules.",
        publisher: { "@id": "https://kube-downscaler.io/#organization" },
      }),
    },
    {
      tagName: "link",
      attributes: {
        rel: "manifest",
        href: "/manifest.json",
      },
    },
    // Organization schema
    {
      tagName: "script",
      attributes: {
        type: "application/ld+json",
      },
      innerHTML: JSON.stringify({
        "@context": "https://schema.org",
        "@type": "Organization",
        "@id": "https://kube-downscaler.io/#organization",
        name: "CaaS Team",
        url: "https://github.com/caas-team",
        logo: "https://kube-downscaler.io/img/kubedownscaler.svg",
        sameAs: ["https://github.com/caas-team"],
      }),
    },
  ],
  themes: [
    [
      require.resolve("@easyops-cn/docusaurus-search-local"),
      {
        hashed: true,
        indexBlog: true,
        docsRouteBasePath: ["/docs", "/guides", "/contributing"],
        docsDir: "content",
        searchBarShortcutHint: false,
      } as Partial<PluginOptions>,
    ],
    "@docusaurus/theme-mermaid",
  ],
  plugins: [
    tailwindPlugin,
    [
      firstDocRedirectPlugin,
      { sidebarConfig: "sidebars.ts" } satisfies firstDocRedirectConfig,
    ],
  ],
  markdown: {
    mermaid: true,
    parseFrontMatter: globalRefParseFrontMatter,
    hooks: {
      onBrokenMarkdownLinks: "throw",
    },
  },
};

export default config;
