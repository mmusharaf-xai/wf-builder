"use client";
import GlobalLayout from "@/components/globals/GlobalLayout";
import React, { useEffect, useState } from "react";
import WorkflowDetailsHeader from "../components/workflowDetailsHeader";
import { ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import {
  Background,
  Controls,
  ReactFlow,
  ReactFlowProvider,
} from "@xyflow/react";
import CommonNode from "../components/CommonNode";
import CustomEdge from "../components/CustomEdge";
import FitViewOnLoad from "../components/FitViewOnLoad";
import LeftPanel from "./LeftPanel";
import { useWorkflowStore } from "@/app/store";
import { apiFetch } from "@/lib/api";
import { useParams, useSearchParams } from "next/navigation";
import LoadingSpinner from "@/components/loaders/SpinnerLoader";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import { AllNodesI, LinkI } from "@/lib/types";

type SectionKey = "HISTORY" | "GRAPH";

const SECTIONS: { key: SectionKey; label: string }[] = [
  { key: "HISTORY", label: "History" },
  { key: "GRAPH", label: "Graph" },
];

function ExecutionGraphCanvas({
  executionId,
  nodes,
  edges,
}: {
  executionId: string;
  nodes: AllNodesI[];
  edges: LinkI[];
}) {
  return (
    // key remounts a clean store so fitView always re-runs for the selected execution
    <ReactFlowProvider key={executionId}>
      <div className="h-full w-full">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          edgeTypes={{
            default: CustomEdge,
          }}
          nodeTypes={{
            WEBHOOK_NODE: CommonNode,
            WEBHOOK_RESPONSE_NODE: CommonNode,
            CODE_NODE: CommonNode,
          }}
          // Built-in init fit after node measure (same path as Controls once dimensions exist)
          fitView
          minZoom={0.15}
          maxZoom={2}
          proOptions={{ hideAttribution: true }}
          className="!h-full !w-full"
        >
          <FitViewOnLoad graphKey={executionId} enabled />
          <Background />
          <Controls
            position="top-left"
            className="!m-2 sm:!m-3 scale-90 sm:scale-100"
          />
        </ReactFlow>
      </div>
    </ReactFlowProvider>
  );
}

function ExecutionGraph({
  executionId,
  visible,
}: {
  executionId: string;
  visible: boolean;
}) {
  const {
    executionState: { detailLoading, detailError, executionsDetails },
  } = useWorkflowStore();

  if (detailError) {
    return (
      <div className="absolute inset-0 flex items-center justify-center p-4 text-center text-sm text-red-500 font-medium">
        {detailError}
      </div>
    );
  }

  if (detailLoading) {
    return <LoadingSpinner isLoading />;
  }

  if (!executionsDetails.nodes.length) {
    return (
      <div className="absolute inset-0 flex items-center justify-center p-4 text-center text-sm text-muted-foreground font-medium">
        No nodes in this execution.
      </div>
    );
  }

  // Only mount when panel is on-screen so width/height are non-zero for fitView.
  if (!visible) {
    return null;
  }

  return (
    <ExecutionGraphCanvas
      executionId={executionId}
      nodes={executionsDetails.nodes}
      edges={executionsDetails.edges}
    />
  );
}

function ExecutionsPage() {
  const { updateExecutionState, update } = useWorkflowStore();
  const searchParams = useSearchParams();
  const executionId = searchParams.get("e_id");
  const { id } = useParams();
  const isDesktop = useMediaQuery("(min-width: 768px)");
  const [mobileSection, setMobileSection] = useState<SectionKey>(
    executionId ? "GRAPH" : "HISTORY"
  );

  useEffect(() => {
    if (!isDesktop && executionId) {
      setMobileSection("GRAPH");
    }
  }, [executionId, isDesktop]);

  useEffect(() => {
    update({
      nodesData: {},
    });
    updateExecutionState({
      detailError: "",
      executionsDetails: {
        edges: [],
        nodes: [],
      },
    });
    if (executionId && id) {
      void getWorkflowHistoryDetails();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [executionId, id]);

  const getWorkflowHistoryDetails = async () => {
    try {
      updateExecutionState({
        detailLoading: true,
      });
      const response = await apiFetch(
        `/api/workflows/${id}/executions/${executionId}`
      );
      const data = await response.json();
      if (data?.error === false) {
        updateExecutionState({
          detailLoading: false,
          executionsDetails: {
            edges: data?.edges || [],
            nodes: data?.nodes || [],
          },
        });
      } else if (data?.message) {
        updateExecutionState({
          detailLoading: false,
          detailError: data?.message,
          executionsDetails: {
            edges: [],
            nodes: [],
          },
        });
      } else {
        throw new Error("Something went wrong");
      }
    } catch (error) {
      console.log(error);
      updateExecutionState({
        detailError: "Something went wrong.Please try again later.",
        detailLoading: false,
        executionsDetails: {
          edges: [],
          nodes: [],
        },
      });
    }
  };

  const historyPanel = (
    <div className="h-full min-h-0 w-full overflow-hidden">
      <LeftPanel />
    </div>
  );

  const graphVisible = isDesktop || mobileSection === "GRAPH";

  const graphPanel = (
    <div className="h-full min-h-0 w-full relative border-l md:border-l">
      {!executionId ? (
        <div className="absolute inset-0 flex items-center justify-center p-4 text-center text-sm text-muted-foreground font-medium">
          Select an execution from History to view the graph.
        </div>
      ) : (
        <ExecutionGraph executionId={executionId} visible={graphVisible} />
      )}
    </div>
  );

  return (
    <div className="h-full w-full flex flex-col min-h-0">
      <WorkflowDetailsHeader />
      <GlobalLayout className="!p-0 flex-1 min-h-0 !h-full !overflow-hidden">
        {!isDesktop ? (
          <div className="flex h-full min-h-0 flex-col overflow-hidden">
            <div
              role="tablist"
              aria-label="Executions sections"
              className="shrink-0 grid grid-cols-2 gap-1 p-1.5 border-b bg-muted/40"
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
              {mobileSection === "HISTORY" ? historyPanel : graphPanel}
            </div>
          </div>
        ) : (
          <ResizablePanelGroup direction="horizontal" className="h-full">
            <ResizablePanel defaultSize={25} minSize={18} maxSize={45}>
              {historyPanel}
            </ResizablePanel>
            <ResizablePanel className="relative" defaultSize={75} minSize={40}>
              {graphPanel}
            </ResizablePanel>
          </ResizablePanelGroup>
        )}
      </GlobalLayout>
    </div>
  );
}

export default ExecutionsPage;
