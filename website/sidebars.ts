import type { SidebarsConfig } from "@docusaurus/plugin-content-docs";

const sidebars: SidebarsConfig = {
  DocsSidebar: [
    {
      type: "autogenerated",
      dirName: "docs",
    },
  ],
  GuidesSidebar: [
    {
      type: "category",
      label: "Getting Started",
      link: {
        type: "doc",
        id: "guides/getting-started",
      },
      items: [
        {
          type: "autogenerated",
          dirName: "guides/getting-started",
        },
      ],
    },
  ],
};

export default sidebars;