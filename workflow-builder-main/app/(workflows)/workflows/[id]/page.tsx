"use client";
import React, { useCallback, useRef, useState } from "react";
import WorkflowDetailsHeader from "./components/workflowDetailsHeader";
import GlobalLayout from "@/components/globals/GlobalLayout";
import {
  addEdge,
  applyEdgeChanges,
  applyNodeChanges,
  Background,
  Connection,
  Controls,
  Edge,
  EdgeChange,
  NodeChange,
  ReactFlow,
  ReactFlowInstance,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { PlusSquare } from "lucide-react";
import GlobalDrawer from "@/components/globals/GlobalDrawer";
import NodesSidebar from "./components/NodesSidebar";
import FitViewOnLoad from "./components/FitViewOnLoad";
import { useDrawer } from "@/app/providers/drawerProvider";
import { AllNodesI, TNodeTypes } from "@/lib/types";
import { toast } from "sonner";
import CommonNode from "./components/CommonNode";
import { useWorkflowStore } from "@/app/store";
import { apiFetch } from "@/lib/api";
import CustomEdge from "./components/CustomEdge";
import { useParams } from "next/navigation";
import LoadingSpinner from "@/components/loaders/SpinnerLoader";

function EditorPage() {
  
  const {
    draftState,
    addNode,
    updateNodes,
    update,
    workflowDetails,
    loading,
    error,
  } = useWorkflowStore();

  const { id } = useParams();
  const { setOpen, setClose } = useDrawer();

  type FlowInstance = ReactFlowInstance<AllNodesI, Edge>;

  const [reactFlowInstance, setReactFlowInstance] = useState<FlowInstance>();
  const addingRef = useRef(false);

  /** Shared create path for drag-drop and click-from-list. */
  const createNodeAt = useCallback(
    async (type: TNodeTypes, position: { x: number; y: number }) => {
      if (!type || !id || addingRef.current) return false;

      let label = `${type.replaceAll("_", " ")}`;
      const triggerAlreadyExists = draftState.nodes.find(
        (node) =>
          ["WEBHOOK_NODE", "WEBHOOK_RESPONSE_NODE"].includes(type) &&
          node.type === type
      );

      if (triggerAlreadyExists) {
        toast.error(`Only one ${label} node is allowed`);
        return false;
      }

      const count = draftState.nodes.reduce(
        (a, b) => a + (b.type === type ? 1 : 0),
        0
      );
      if (count > 0) {
        label = `${label} ${count + 1}`;
      }

      addingRef.current = true;
      try {
        const response = await apiFetch(`/api/workflows/${id}/addNode`, {
          method: "POST",
          body: JSON.stringify({
            label,
            description: "",
            type,
            positionX: position.x,
            positionY: position.y,
          }),
        });

        const json = await response.json();
        if (json?.error === false) {
          const node: AllNodesI = {
            id: json.node.id,
            type,
            position,
            data: {
              label,
              description: "",
            },
          };
          addNode(node);
          return true;
        }
        throw new Error(json.message || "Failed to add node");
      } catch (e) {
        console.log(e);
        toast.error("Failed to add node");
        return false;
      } finally {
        addingRef.current = false;
      }
    },
    [id, draftState.nodes, addNode]
  );

  /** Click-to-add: place near viewport center, offset if nodes already exist. */
  const onSelectNodeType = useCallback(
    async (type: TNodeTypes) => {
      let position = { x: 250, y: 150 };
      if (reactFlowInstance) {
        const bounds = document
          .querySelector(".react-flow")
          ?.getBoundingClientRect();
        if (bounds) {
          position = reactFlowInstance.screenToFlowPosition({
            x: bounds.left + bounds.width / 2,
            y: bounds.top + bounds.height / 2,
          });
        }
        // Stagger multiple click-adds so nodes don't stack exactly on top
        const sameTypeCount = draftState.nodes.filter((n) => n.type === type).length;
        position = {
          x: position.x + sameTypeCount * 40,
          y: position.y + sameTypeCount * 40,
        };
      } else if (draftState.nodes.length > 0) {
        const last = draftState.nodes[draftState.nodes.length - 1];
        position = { x: last.position.x + 180, y: last.position.y };
      }

      const ok = await createNodeAt(type, position);
      if (ok) {
        setClose();
      }
    },
    [reactFlowInstance, draftState.nodes, createNodeAt, setClose]
  );

  const handleClick = () => {
    setOpen(
      <GlobalDrawer
        title="Workflow Nodes"
        subheading="Nodes are the Building blocks for the workflow. Click or drag onto the canvas."
        modal={false}
        hideFooter
      >
        <NodesSidebar onSelectNode={onSelectNodeType} />
      </GlobalDrawer>
    );
  };

  const onDrop = useCallback(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    async (event: any) => {
      event.preventDefault();
      try {
        const type: TNodeTypes = event.dataTransfer.getData(
          "application/reactflow"
        );

        if (typeof type === "undefined" || !type) {
          return;
        }
        if (!reactFlowInstance) return;

        const position = reactFlowInstance.screenToFlowPosition({
          x: event.clientX,
          y: event.clientY,
        });
        await createNodeAt(type, position);
      } catch (e) {
        console.log(e);
        toast.error("Failed to add node");
      }
    },
    [reactFlowInstance, createNodeAt]
  );

  const onNodesChange = useCallback(
    (changes: NodeChange[]) => {
      updateNodes(applyNodeChanges(changes, draftState.nodes) as AllNodesI[]);
    },
    [draftState.nodes]
  );

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const onDragOver = useCallback((event: any) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
  }, []);

  const onEdgesChange = useCallback(
    (changes: EdgeChange[]) => {
      update({
        showSave: true,
        draftState: {
          ...draftState,
          edges: applyEdgeChanges(changes, draftState.edges),
        },
      });
    },
    [draftState]
  );

  const onConnect = useCallback(
    (params: Edge | Connection) => {
      update({
        showSave: true,
        draftState: {
          ...draftState,
          edges: addEdge({ ...params, type: "default" }, draftState.edges),
        },
      });
    },
    [draftState]
  );

  return (
    <>
      <WorkflowDetailsHeader />
      <GlobalLayout className="!p-0">
        <div className="relative h-full min-h-[50vh] w-full">
          <LoadingSpinner isLoading={loading} />
          {workflowDetails?.id && !error ? (
            <button
              type="button"
              aria-label="Add node"
              className="absolute rounded-lg cursor-pointer border shadow-md p-2 top-2 right-2 sm:right-4 z-50 bg-background hover:bg-muted touch-manipulation"
              onClick={(e) => {
                e.stopPropagation();
                handleClick();
              }}
            >
              <PlusSquare size={28} className="sm:hidden" />
              <PlusSquare size={40} className="hidden sm:block" />
            </button>
          ) : null}

          <ReactFlow
            onEdgesChange={onEdgesChange}
            onNodesChange={onNodesChange}
            onDrop={onDrop}
            onConnect={onConnect}
            onDragOver={onDragOver}
            onInit={setReactFlowInstance}
            nodes={draftState.nodes}
            edgeTypes={{
              default: CustomEdge,
            }}
            edges={draftState.edges}
            nodeTypes={{
              WEBHOOK_NODE: CommonNode,
              WEBHOOK_RESPONSE_NODE: CommonNode,
              CODE_NODE: CommonNode,
            }}
            proOptions={{ hideAttribution: true }}
            className="!h-full !w-full"
          >
            {/* Same fitView() as Controls icon, after node sizes are measured */}
            <FitViewOnLoad
              graphKey={String(workflowDetails?.id || id || "")}
              enabled={!loading && !error && !!workflowDetails?.id}
            />
            <Background />
            <Controls position="top-left" className="!m-2 sm:!m-3 scale-90 sm:scale-100" />
          </ReactFlow>
        </div>
      </GlobalLayout>
    </>
  );
}

export default EditorPage;
