import React, { useEffect, useState } from "react";
import { useColorMode } from "@docusaurus/theme-common";
import { exportToSvg } from "@excalidraw/utils";
import type { ExcalidrawRendererProps } from "@site/src/types/components/common";

export const ExcalidrawRenderer: React.FC<ExcalidrawRendererProps> = ({
  data,
  className,
}) => {
  const [svgElement, setSvgElement] = useState<SVGSVGElement | null>(null);
  const { colorMode } = useColorMode();

  useEffect(() => {
    const generateStaticSvg = async () => {
      const result = await exportToSvg({
        elements: data.elements,
        files: data.files,
        appState: {
          ...data.appState,
          exportWithDarkMode: colorMode == "dark",
          exportBackground: false,
        },
        exportPadding: 30,
      });

      result.removeAttribute("height");
      result.removeAttribute("width");

      setSvgElement(result);
    };
    generateStaticSvg();
  }, [data, colorMode]);

  if (!svgElement) return <>Loading...</>;
  return (
    <div
      className={`select-none ${className}`}
      dangerouslySetInnerHTML={{ __html: svgElement.outerHTML }}
    />
  );
};
