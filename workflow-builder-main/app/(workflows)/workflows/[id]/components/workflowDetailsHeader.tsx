"use client";
import { useWorkflowStore } from "@/app/store";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import Link from "next/link";
import { usePathname } from "next/navigation";
import React, { useMemo, useState } from "react";

import { toast } from "sonner";
import { apiFetch } from "@/lib/api";

const WorkflowDetailsHeader = () => {
  const pathname = usePathname();
  const { loading, workflowDetails, showSave, update, draftState } =
    useWorkflowStore();

  const [isSaving, setisSaving] = useState(false);

  const onSave = async () => {
    try {
      setisSaving(true);
      const response = await apiFetch(
        `/api/workflows/${workflowDetails?.id}/updateNodeAndEdges`,
        {
          method: "PUT",
          body: JSON.stringify({
            nodes: draftState.nodes,
            edges: draftState.edges,
          }),
        }
      );
      const data = await response.json();
      if (data?.error === false) {
        toast.message(data.message);
        update({ showSave: false, mainState: draftState });
      } else {
        if (data?.message) {
          toast.error(data.message);
        } else {
          throw new Error("Something went wrong");
        }
      }
    } catch (e) {
      console.log(e);
      toast.error("Failed to Save");
    } finally {
      setisSaving(false);
    }
  };
  const isExecugtionPage = useMemo(
    () => pathname?.includes("executions"),
    [pathname]
  );
  return (
    <div className="flex flex-col sm:flex-row gap-3 sm:gap-6 min-h-14 sm:min-h-16 items-stretch sm:items-center px-3 sm:px-6 py-3 sm:py-4 w-full dark:bg-black">
      <div className="flex items-center justify-between gap-2 sm:flex-1 min-w-0">
        {!workflowDetails?.name && loading ? (
          <Skeleton className="w-full max-w-[15rem] h-8" />
        ) : (
          <h1
            className="font-semibold text-base sm:text-lg max-w-[70vw] sm:max-w-[20rem] truncate"
            title={workflowDetails?.name || ""}
          >
            {workflowDetails?.name || "Something went wrong"}
          </h1>
        )}
        <div className="sm:hidden shrink-0">
          {isExecugtionPage ? null : loading || isSaving ? (
            <Skeleton className="w-14 h-8" />
          ) : showSave ? (
            <Button size={"sm"} onClick={onSave}>
              Save
            </Button>
          ) : null}
        </div>
      </div>
      <div className="flex justify-center items-center sm:flex-1">
        <Tabs
          defaultValue="account"
          value={isExecugtionPage ? "executions" : "editor"}
          className="w-full sm:w-auto"
        >
          <TabsList className="w-full sm:w-auto">
            <Link href={`/workflows/${workflowDetails?.id}`} className="flex-1 sm:flex-initial">
              <TabsTrigger value="editor" className="w-full">
                Editor
              </TabsTrigger>
            </Link>
            <Link
              href={`/workflows/${workflowDetails?.id}/executions`}
              className="flex-1 sm:flex-initial"
            >
              <TabsTrigger value="executions" className="w-full">
                Executions
              </TabsTrigger>
            </Link>
          </TabsList>
        </Tabs>
      </div>
      <div className="hidden sm:flex flex-1 items-center min-w-[5rem] gap-2 justify-end">
        {isExecugtionPage ? null : loading || isSaving ? (
          <Skeleton className="w-full max-w-[5rem] h-8" />
        ) : showSave ? (
          <Button size={"sm"} onClick={onSave}>
            Save
          </Button>
        ) : null}
      </div>
    </div>
  );
};

export default WorkflowDetailsHeader;
