import useDocusaurusContext from "@docusaurus/useDocusaurusContext";
import React, { useEffect, useState } from "react";
import { hexToRGB, rgbToHSL } from "../util/color";
import { useGithub } from "./hook/githubHook";

export const GithubLabel: React.FC<GithubLabelProps> = ({
  label: labelName,
}) => {
  const { labels } = useGithub();
  const [label, setLabel] = useState<CachedGithubLabel>(undefined);

  useEffect(() => {
    if (!labels) return;
    setLabel(labels.find((l) => l.name === labelName));
  }, [labels]);

  return label ? (
    <a
      title={label.description}
      href={label.url}
      target="_blank"
      className="px-2.5 py-0.5 border border-solid rounded-3xl text-sm font-medium whitespace-nowrap cursor-pointer no-underline hover:no-underline"
      style={{
        color: `hsl(${label.lightHsl.h}, ${label.lightHsl.s}%, ${label.lightHsl.l}%)`,
        background: `rgba(${label.rgb.r}, ${label.rgb.g}, ${label.rgb.b}, 18%)`,
        borderColor: `hsla(${label.lightHsl.h}, ${label.lightHsl.s}%, ${label.lightHsl.l}%, 30%)`,
      }}
    >
      {labelName}
    </a>
  ) : (
    <code>{labelName}</code>
  );
};
