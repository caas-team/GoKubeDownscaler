import type {
  ExcalidrawElement,
  NonDeleted,
} from "@excalidraw/utils/dist/excalidraw/element/types";
import type {
  AppState,
  BinaryFiles,
} from "@excalidraw/utils/dist/excalidraw/types";

interface GithubLabelProps {
  label: string;
}

interface ExcalidrawRendererProps {
  data: {
    elements: readonly NonDeleted<ExcalidrawElement>[];
    appState?: Partial<Omit<AppState, "offsetTop" | "offsetLeft">>;
    files: BinaryFiles | null;
  };
  className: string;
}
