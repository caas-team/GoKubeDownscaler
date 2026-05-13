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
                  // prefix ids and class names with the filename, to prevent duplicate ids from interfering with each other
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
          ignorePatterns: [
              "/tags/**",
              "/docs/v*/**",
              "/docs/next/**",
              "/guides/v*/**",
              "/guides/next/**",
              "/contributing/v*/**",
              "/contributing/next/**",
          ],
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
        href: "/",
        target: "_self",
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
    // Default favicon
    {
      tagName: "link",
      attributes: {
        rel: "icon",
        type: "image/svg+xml",
        href: "/img/kubedownscaler.svg",
      },
    },
    // PNG 16x16 favicon
    {
      tagName: "link",
      attributes: {
        rel: "icon",
        type: "image/png",
        sizes: "16x16",
        href: "/img/kubedownscaler-16x16.png",
      },
    },
    // PNG 32x32 favicon
    {
      tagName: "link",
      attributes: {
        rel: "icon",
        type: "image/png",
        sizes: "32x32",
        href: "/img/kubedownscaler-32x32.png",
      },
    },
    // PNG 48x48 favicon
    {
      tagName: "link",
      attributes: {
        rel: "icon",
        type: "image/png",
        sizes: "48x48",
        href: "/img/kubedownscaler-48x48.png",
      },
    },
    // Apple Touch Icon (iOS) favicon
    {
      tagName: "link",
      attributes: {
        rel: "apple-touch-icon",
        sizes: "180x180",
        href: "/img/kubedownscaler-180x180.png",
      },
    },
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
        description: "Open-source team maintaining GoKubeDownscaler and related Kubernetes tooling.",
        url: "https://github.com/caas-team",
        sameAs: ["https://github.com/caas-team"],
        logo: {
          "@type": "ImageObject",
          url: "https://kube-downscaler.io/img/kubedownscaler.svg",
          width: "512",
          height: "512"
        },
        contactPoint: {
          "@type": "ContactPoint",
          contactType: "community support",
          url: "https://inviter.co/kube-downscaler"
        }
      }),
    },
    // Website schema
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
        description: "Official website with documentation and guides for GoKubeDownscaler, a Kubernetes horizontal autoscaler that reduces cloud costs by scaling workloads based on time schedules.",
        inLanguage: "en",
        publisher: { "@id": "https://kube-downscaler.io/#organization" },
        mainEntity: { "@id": "https://kube-downscaler.io/#software" }
      }),
    },
    // SoftwareApplication
    {
      tagName: "script",
      attributes: {
        type: "application/ld+json",
      },
      innerHTML: JSON.stringify({
        "@context": "https://schema.org",
        "@type": "SoftwareApplication",
        "@id": "https://kube-downscaler.io/#software",
        name: "GoKubeDownscaler",
        alternateName: ["kube-downscaler", "go-kube-downscaler", "kubernetes downscaler"],
        about: [
          {
            "@type": "Thing",
            name: "Kubernetes"
          },
          {
            "@type": "Thing",
            name: "Autoscaling"
          },
          {
            "@type": "Thing",
            name: "Cloud Cost Optimization"
          }
        ],
        description: "GoKubeDownscaler is a horizontal autoscaler that scales Kubernetes workloads down during off-hours (nights, weekends, holidays) to reduce cloud costs.",
        image: "https://kube-downscaler.io/img/social-preview.png",
        url: "https://kube-downscaler.io/",
        applicationCategory: "DeveloperApplication",
        applicationSubCategory: "Kubernetes Addon",
        operatingSystem: "Linux",
        softwareRequirements: "Kubernetes >= 1.23",
        downloadUrl: "https://github.com/caas-team/GoKubeDownscaler/releases/latest",
        installUrl: "https://kube-downscaler.io/docs/getting-started",
        releaseNotes: "https://github.com/caas-team/GoKubeDownscaler/releases",
        codeRepository: "https://github.com/caas-team/GoKubeDownscaler",
        license: "https://www.gnu.org/licenses/gpl-3.0.en.html",
        discussionUrl: "https://inviter.co/kube-downscaler",
        bugTrackerUrl: "https://github.com/caas-team/GoKubeDownscaler/issues",
        programmingLanguage: {
          "@type": "ComputerLanguage",
          name: "Go",
          url: "https://go.dev"
        },
        featureList: [
          "Scheduled scale-down for Kubernetes workloads during off-hours",
          "Namespace- and workload-level annotation overrides",
          "Recurring schedules and RFC3339 time windows",
          "Cloud-agnostic support for AWS, GCP, Azure, and on-premises Kubernetes",
          "Integrates with KEDA, Prometheus, Argo, and other CRDs",
          "One-command Helm chart installation and upgrades"
        ],
        screenshot: "https://kube-downscaler.io/img/social-preview.png",
        offers: {
          "@type": "Offer",
          price: "0",
          priceCurrency: "USD"
        },
        keywords: [
          "Kubernetes",
          "autoscaling",
          "scheduled scaling",
          "cost optimization",
          "DevOps",
          "cloud infrastructure"
        ],
        author: { "@id": "https://kube-downscaler.io/#organization" },
        publisher: { "@id": "https://kube-downscaler.io/#organization" },
        maintainer: { "@id": "https://kube-downscaler.io/#organization" },
        sameAs: [
          "https://github.com/caas-team/GoKubeDownscaler",
          "https://artifacthub.io/packages/helm/py-kube-downscaler/go-kube-downscaler",
          "https://www.producthunt.com/products/gokubedownscaler",
        ],
        hasPart: { "@id": "https://kube-downscaler.io/#source-code" }
      }),
    },
    {
      tagName: "link",
      attributes: {
        rel: "manifest",
        href: "/manifest.json",
      },
    },
    {
      tagName: "script",
      attributes: {
        type: "application/ld+json",
      },
      innerHTML: JSON.stringify({
        "@context": "https://schema.org",
        "@type": "SoftwareSourceCode",
        "@id": "https://kube-downscaler.io/#source-code",
        name: "GoKubeDownscaler Source Code",
        description: "Source code repository for GoKubeDownscaler, written in Go.",
        url: "https://github.com/caas-team/GoKubeDownscaler",
        codeRepository: "https://github.com/caas-team/GoKubeDownscaler",
        codeSampleType: "full solution",
        programmingLanguage: {
          "@type": "ComputerLanguage",
          name: "Go",
          url: "https://go.dev"
        },
        runtimePlatform: "Kubernetes",
        license: "https://www.gnu.org/licenses/gpl-3.0.en.html",
        author: { "@id": "https://kube-downscaler.io/#organization" },
        publisher: { "@id": "https://kube-downscaler.io/#organization" },
        isBasedOn: { "@id": "https://kube-downscaler.io/#py-kube-downscaler-code" },
        isPartOf: { "@id": "https://kube-downscaler.io/#software" }
      }),
    },
    // py-kube-downscaler source code (predecessor)
    {
      tagName: "script",
      attributes: {
        type: "application/ld+json",
      },
      innerHTML: JSON.stringify({
        "@context": "https://schema.org",
        "@type": "SoftwareSourceCode",
        "@id": "https://kube-downscaler.io/#py-kube-downscaler-code",
        name: "py-kube-downscaler",
        description: "Python fork of the original kube-downscaler. Predecessor to GoKubeDownscaler.",
        url: "https://github.com/caas-team/py-kube-downscaler",
        codeRepository: "https://github.com/caas-team/py-kube-downscaler",
        programmingLanguage: {
          "@type": "ComputerLanguage",
          name: "Python",
          url: "https://www.python.org"
        },
        license: "https://www.gnu.org/licenses/gpl-3.0.en.html",
        author: { "@id": "https://kube-downscaler.io/#organization" },
        isBasedOn: { "@id": "https://kube-downscaler.io/#original-kube-downscaler-code" }
      }),
    },
    // Original kube-downscaler source code (original, no longer maintained)
    {
      tagName: "script",
      attributes: {
        type: "application/ld+json",
      },
      innerHTML: JSON.stringify({
        "@context": "https://schema.org",
        "@type": "SoftwareSourceCode",
        "@id": "https://kube-downscaler.io/#original-kube-downscaler-code",
        name: "kube-downscaler (Original)",
        description: "Original kube-downscaler project by Henning Jacobs. Foundation for GoKubeDownscaler.",
        url: "https://codeberg.org/hjacobs/kube-downscaler",
        codeRepository: "https://codeberg.org/hjacobs/kube-downscaler",
        programmingLanguage: {
          "@type": "ComputerLanguage",
          name: "Python",
          url: "https://www.python.org"
        },
        license: "https://www.gnu.org/licenses/gpl-3.0.en.html",
      }),
    },
    // Service schema - describes the capability/service the software provides
    {
      tagName: "script",
      attributes: {
        type: "application/ld+json",
      },
      innerHTML: JSON.stringify({
        "@context": "https://schema.org",
        "@type": "Service",
        "@id": "https://kube-downscaler.io/#service",
        name: "Kubernetes Scheduled Autoscaling for Cost Reduction",
        alternateName: [
          "Kubernetes off-hours autoscaling",
          "scheduled Kubernetes scaling",
          "Kubernetes cost optimization service"
        ],
        serviceType: "Kubernetes Cost Optimization",
        category: "Devops Tool",
        description: "A Kubernetes-native autoscaler that scales workloads to zero during off-hours to reduce cloud costs",
        mainEntityOfPage: {
          "@type": "WebPage",
          "@id": "https://kube-downscaler.io/"
        },
        keywords: [
          "Kubernetes autoscaling",
          "scheduled workload scaling",
          "Kubernetes cost optimization",
          "off-hours scaling"
        ],
        audience: [
          {
            "@type": "Audience",
            audienceType: "DevOps Engineer",
          },
          {
            "@type": "Audience",
            audienceType: "Platform Engineer",
          },
          {
            "@type": "Audience",
            audienceType: "Site Reliability Engineer (SRE)",
          },
          {
            "@type": "Audience",
            audienceType: "Cloud Architect",
          },
          {
            "@type": "Audience",
            audienceType: "Infrastructure Team Lead",
          },
          {
            "@type": "Audience",
            audienceType: "Kubernetes Administrator",
          },
        ],
        provider: {
          "@id": "https://kube-downscaler.io/#organization"
        },
        about: {
          "@id": "https://kube-downscaler.io/#software"
        },
        serviceOutput: {
          "@type": "Thing",
          name: "Reduced Kubernetes compute cost via scheduled scale-down during off-hours, weekends and holidays"
        },
        potentialAction: {
          "@type": "Action",
          name: "Configure scheduled Kubernetes autoscaling",
          description: "Define time-based rules to automatically scale Kubernetes workloads during off-hours, weekends, or low-traffic periods to reduce infrastructure costs.",
          target: {
            "@type": "EntryPoint",
            urlTemplate: "https://kube-downscaler.io/docs/getting-started",
            actionPlatform: [
              "http://schema.org/DesktopWebPlatform"
            ]
          }
        }
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
