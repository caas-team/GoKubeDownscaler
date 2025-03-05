import React, { useEffect, useRef } from "react";
import { exportToSvg } from "@excalidraw/excalidraw";
import { ImportedDataState } from "@excalidraw/excalidraw/types/data/types";
import { useColorMode } from "@docusaurus/theme-common";

export const ExcalidrawWrapper: React.FC<ExcalidrawWrapperProps> = ({
  data,
}) => {
  const svg = useRef<HTMLDivElement>(null);
  const { colorMode } = useColorMode();

  useEffect(() => {
    const generateStaticSvg = async () => {
      const result = await exportToSvg({
        ...(data as Required<ImportedDataState>),
        appState: {
          exportWithDarkMode: colorMode == "dark",
          exportBackground: false,
        },
        exportPadding: 30,
      });

      result.removeAttribute("height");
      result.removeAttribute("width");

      if (!svg.current) return;
      svg.current.appendChild(result);
    };
    generateStaticSvg();

    return () => {
      if (svg.current) {
        svg.current.removeChild(svg.current.firstChild);
      }
    };
  }, [data, colorMode, svg]);

  return <div className="select-none" ref={svg} />;
};
