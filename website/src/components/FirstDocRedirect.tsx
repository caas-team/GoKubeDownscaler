import React from "react";
import { Redirect } from "@docusaurus/router";
import {
  useActivePlugin,
  useAllDocsData,
} from "@docusaurus/plugin-content-docs/client";
import useDocusaurusContext from "@docusaurus/useDocusaurusContext";

interface RedirectToFirstDocProps {
  sidebar: string;
}

export const RedirectToFirstDoc: React.FC<RedirectToFirstDocProps> = ({
  sidebar,
}) => {
  const activePlugin = useActivePlugin();
  const allDocsData = useAllDocsData();
  const { siteConfig } = useDocusaurusContext();

  if (!activePlugin || !allDocsData) {
    return null; // wait
  }

  const pluginId = activePlugin.pluginId;
  const version = allDocsData[pluginId]?.versions[0];

  if (!version) {
    console.error("No version data available");
    return <Redirect to={siteConfig.baseUrl} />;
  }

  const sidebarItem = version.sidebars[sidebar];

  if (!sidebarItem) {
    console.error(`Sidebar "${sidebar}" does not exist`);
    return <Redirect to={siteConfig.baseUrl} />;
  }

  if (sidebarItem.link?.path) {
    return <Redirect to={sidebarItem.link.path} />;
  }

  console.error(`Sidebar "${sidebar}" does not contain a valid link`);
  return <Redirect to={siteConfig.baseUrl} />;
};
