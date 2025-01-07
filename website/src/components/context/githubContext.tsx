import useDocusaurusContext from "@docusaurus/useDocusaurusContext";
import { FetchGithubLabels } from "@site/src/api/github";
import React, { createContext, useEffect, useState } from "react";

export const GithubContext = createContext<IGithubContext>(undefined);

export const GithubProvider: React.FC<IContextProps> = ({ children }) => {
  const {siteConfig: { organizationName, projectName }} = useDocusaurusContext(); // prettier-ignore
  const [labelData, setLabelData] = useState<CachedGithubLabel[]>(undefined);

  useEffect(() => {
    // get static assets from github on build
    const fetchLabels = async () => {
      try {
        const labels = await FetchGithubLabels({
          org: organizationName,
          repo: projectName,
        });
        setLabelData(labels);
      } catch (error) {
        console.error(`Failed to fetch labels: ${error.message}`);
      }
    };

    fetchLabels();
  }, [organizationName, projectName]);

  return (
    <GithubContext.Provider value={{ labels: labelData }}>
      {children}
    </GithubContext.Provider>
  );
};
