"use server";

import { apiFetch } from "@/lib/api";
import {
  GetWorkflowsParams,
  GetWorkflowsResponse,
  Workflow,
} from "@/lib/types";
import { fetchWorkflows, WORKFLOWS_PAGE_SIZE } from "@/lib/workflows-api";

/**
 * Workflow list/create/update/delete — backed by the Go API.
 */

export const onCreateWorkflow = async (name: string, description: string) => {
  try {
    const response = await apiFetch("/api/workflows", {
      method: "POST",
      body: JSON.stringify({ name, description }),
      cache: "no-store",
    });
    const data = await response.json();
    if (!response.ok || data?.error) {
      return { message: data?.message || "Oops! try again", error: true as const };
    }
    return {
      message: data?.message || "workflow created",
      workflow: data?.workflow as Workflow | undefined,
    };
  } catch (error) {
    console.error("Error creating workflow:", error);
    return { message: "Oops! try again", error: true as const };
  }
};

export const onUpdateWorkflow = async (workflowData: Partial<Workflow>) => {
  if (!workflowData?.id) {
    return { message: "Workflow ID is required for updating", error: true as const };
  }

  try {
    const { id, is_deleted: _isDeleted, user_id: _userId, ...rest } = workflowData;
    const response = await apiFetch(`/api/workflows/${id}`, {
      method: "PUT",
      body: JSON.stringify(rest),
      cache: "no-store",
    });
    const data = await response.json();
    if (!response.ok || data?.error) {
      return {
        message: data?.message || "Failed to update workflow",
        error: true as const,
      };
    }
    return {
      message: data?.message || "Workflow updated successfully",
      workflow: data?.workflow as Workflow | undefined,
    };
  } catch (error) {
    console.error("Error updating workflow:", error);
    return {
      message: "Failed to update workflow",
      error: true as const,
    };
  }
};

export const onDeleteWorkflow = async (workflowId: string) => {
  if (!workflowId) {
    return { message: "Workflow ID is required for updating", error: true as const };
  }

  try {
    const response = await apiFetch(`/api/workflows/${workflowId}`, {
      method: "DELETE",
      cache: "no-store",
    });
    const data = await response.json();
    if (!response.ok || data?.error) {
      return {
        message: data?.message || "Failed to update workflow",
        error: true as const,
      };
    }
    return {
      message: data?.message || "Workflow Deleted successfully",
      workflow: data?.workflow as Workflow | undefined,
    };
  } catch (error) {
    console.error("Error deleting workflow:", error);
    return {
      message: "Failed to update workflow",
      error: true as const,
    };
  }
};

export const onGetWorkflows = async (
  params: GetWorkflowsParams = {}
): Promise<GetWorkflowsResponse> => {
  return fetchWorkflows({
    page: 1,
    limit: WORKFLOWS_PAGE_SIZE,
    sortOrder: "desc",
    ...params,
  });
};
