import BrowserOnly from "@docusaurus/BrowserOnly";
import React from "react";

// This acts as a wrapper for ExcalidrawRenderer to not cause SSG/SSR conflicts
const Excalidraw: React.FC<ExcalidrawRendererProps> = (props) => {
  return (
    <BrowserOnly fallback={<>Loading...</>}>
      {() => {
        // eslint-disable-next-line @typescript-eslint/no-require-imports
        const Renderer = require("./ExcalidrawRenderer.tsx").ExcalidrawRenderer;
        return <Renderer {...props} />;
      }}
    </BrowserOnly>
  );
};

export default Excalidraw;
