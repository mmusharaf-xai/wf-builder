import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import React, { useCallback, useMemo, useState } from "react";
import NodeDataView from "./NodeDataView";
import { X, GripVertical } from "lucide-react";
import { useDrawer } from "@/app/providers/drawerProvider";

import Header from "./Header";
import NodeNameEditor from "./NodeNameEditor";
import NodeSections from "./NodeSections";
import NodeInputDataView from "./NodeInputDataView";
import { useNodesEditor } from "../../hooks";
import { useWorkflowStore } from "@/app/store";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";

type SectionKey = "INPUT" | "NODE" | "OUTPUT";

const SECTIONS: { key: SectionKey; label: string }[] = [
  { key: "INPUT", label: "Input" },
  { key: "NODE", label: "Node" },
  { key: "OUTPUT", label: "Output" },
];

const CustomHandle = ({ disabled }: { disabled: boolean }) =>
  disabled ? null : (
    <ResizableHandle disabled={disabled}>
      <div className="h-full w-full flex items-center justify-center">
        <GripVertical
          className={`h-4 w-4 ${
            disabled ? "opacity-0" : "opacity-50 hover:opacity-100"
          }`}
        />
      </div>
    </ResizableHandle>
  );

function NodesEditor() {
  const { setClose } = useDrawer();
  const [fullScreenView, setFullScreenView] = useState("");
  const [mobileSection, setMobileSection] = useState<SectionKey>("NODE");
  const { nodeData } = useNodesEditor();
  const { nodesData } = useWorkflowStore();
  const isDesktop = useMediaQuery("(min-width: 768px)");

  const selectedFullView = useMemo(() => {
    return {
      OUTPUT: fullScreenView === "OUTPUT",
      NODE: fullScreenView === "NODE",
      INPUT: fullScreenView === "INPUT",
    };
  }, [fullScreenView]);

  const onChangeScreen = useCallback((value: SectionKey) => {
    setFullScreenView((prev) => (prev === value ? "" : value));
  }, []);

  const getOutputData = useMemo(() => {
    return nodeData?.id
      ? nodesData?.[nodeData?.id]?.outputData?.map((d) => d.outputJson)
      : "";
  }, [nodesData, nodeData?.id]);

  const inputPanel = (
    <div className="flex flex-col h-full min-h-0 w-full">
      {isDesktop ? (
        <Header
          onChangeFullScreen={() => onChangeScreen("INPUT")}
          title="Input"
          isFullScreen={selectedFullView.INPUT}
        />
      ) : null}
      <div className="flex-1 min-h-0 overflow-auto">
        <NodeInputDataView />
      </div>
    </div>
  );

  const nodePanel = (
    <div className="flex flex-col h-full min-h-0 w-full">
      {isDesktop ? <Header title="Node" /> : null}
      <div className="flex-1 min-h-0 overflow-auto">
        <NodeSections />
      </div>
    </div>
  );

  const outputPanel = (
    <div className="flex flex-col h-full min-h-0 w-full">
      {isDesktop ? (
        <Header
          onChangeFullScreen={() => onChangeScreen("OUTPUT")}
          title="Output"
          isFullScreen={selectedFullView.OUTPUT}
        />
      ) : null}
      <div className="flex-1 min-h-0 overflow-auto">
        <NodeDataView data={getOutputData} />
      </div>
    </div>
  );

  return (
    <main className="flex h-full min-h-0 w-full flex-col gap-3 sm:gap-4 pt-2 sm:pt-3 overflow-hidden">
      <header className="flex items-center gap-2 px-2 flex-wrap shrink-0">
        <div className="flex-1 min-w-0">
          <NodeNameEditor />
        </div>
        <button
          type="button"
          aria-label="Close"
          className="p-1 rounded-md hover:bg-muted touch-manipulation"
          onClick={setClose}
        >
          <X className="h-5 w-5" />
        </button>
      </header>

      {!isDesktop ? (
        <div className="flex-1 min-h-0 flex flex-col border rounded-md overflow-hidden bg-background">
          {/* Section swapper */}
          <div
            role="tablist"
            aria-label="Node editor sections"
            className="shrink-0 grid grid-cols-3 gap-1 p-1.5 border-b bg-muted/40"
          >
            {SECTIONS.map((s) => {
              const active = mobileSection === s.key;
              return (
                <button
                  key={s.key}
                  type="button"
                  role="tab"
                  aria-selected={active}
                  className={cn(
                    "h-9 rounded-md text-xs font-medium touch-manipulation transition-colors",
                    active
                      ? "bg-background text-foreground shadow-sm"
                      : "text-muted-foreground hover:text-foreground"
                  )}
                  onClick={() => setMobileSection(s.key)}
                >
                  {s.label}
                </button>
              );
            })}
          </div>

          <div className="flex-1 min-h-0 overflow-hidden">
            {mobileSection === "INPUT" && inputPanel}
            {mobileSection === "NODE" && nodePanel}
            {mobileSection === "OUTPUT" && outputPanel}
          </div>
        </div>
      ) : (
        <div className="flex-1 min-h-0 w-full border rounded-md overflow-hidden">
          <ResizablePanelGroup direction="horizontal">
            <ResizablePanel
              defaultSize={100}
              minSize={25}
              className={`flex flex-col h-full w-full ${
                fullScreenView && fullScreenView !== "INPUT" ? "hidden" : ""
              }`}
            >
              {inputPanel}
            </ResizablePanel>
            <CustomHandle disabled={!!fullScreenView} />
            <ResizablePanel
              defaultSize={100}
              minSize={25}
              className={`flex flex-col h-full w-full ${
                fullScreenView && fullScreenView !== "NODE" ? "hidden" : ""
              }`}
            >
              {nodePanel}
            </ResizablePanel>
            <CustomHandle disabled={!!fullScreenView} />
            <ResizablePanel
              defaultSize={100}
              minSize={25}
              className={`flex flex-col h-full w-full ${
                fullScreenView && fullScreenView !== "OUTPUT" ? "hidden" : ""
              }`}
            >
              {outputPanel}
            </ResizablePanel>
          </ResizablePanelGroup>
        </div>
      )}
    </main>
  );
}

export default NodesEditor;
