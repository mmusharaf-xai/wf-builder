import { apiFetch } from "@/lib/api";
import { GetWorkflowsParams, GetWorkflowsResponse } from "@/lib/types";

export const WORKFLOWS_PAGE_SIZE = 25;

/** Client/server fetch for paginated, searchable workflow list. */
export async function fetchWorkflows({
  page = 1,
  limit = WORKFLOWS_PAGE_SIZE,
  search = "",
  sortOrder = "desc",
}: GetWorkflowsParams = {}): Promise<GetWorkflowsResponse> {
  const params = new URLSearchParams({
    page: String(page),
    limit: String(limit),
    sortOrder,
  });
  if (search.trim()) {
    params.set("search", search.trim());
  }

  try {
    const response = await apiFetch(`/api/workflows?${params.toString()}`, {
      method: "GET",
      cache: "no-store",
    });
    const data = await response.json();

    if (!response.ok || data?.error) {
      return {
        error:
          typeof data?.error === "string"
            ? data.error
            : data?.message || "Failed to fetch workflows",
        workflows: [],
        metadata: {
          total: 0,
          page: 0,
          totalPages: 0,
          hasNextPage: false,
          limit,
        },
      };
    }

    const metadata = data.metadata ?? {};
    return {
      workflows: data.workflows ?? [],
      metadata: {
        total: metadata.total ?? 0,
        page: metadata.page ?? page,
        totalPages: metadata.totalPages ?? 0,
        hasNextPage: Boolean(metadata.hasNextPage),
        hasPreviousPage: Boolean(metadata.hasPreviousPage),
        limit: metadata.limit ?? limit,
      },
    };
  } catch (error) {
    console.error("Error fetching workflows:", error);
    return {
      error: "Failed to fetch workflows",
      workflows: [],
      metadata: {
        total: 0,
        page: 0,
        totalPages: 0,
        hasNextPage: false,
        limit,
      },
    };
  }
}

export const WORKFLOWS_CHANGED_EVENT = "workflows:changed";

export function notifyWorkflowsChanged() {
  if (typeof window !== "undefined") {
    window.dispatchEvent(new CustomEvent(WORKFLOWS_CHANGED_EVENT));
  }
}
