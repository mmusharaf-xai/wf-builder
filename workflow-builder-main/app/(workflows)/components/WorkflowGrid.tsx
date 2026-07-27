"use client";

import WorkflowCard from "./WorkflowCard";
import { ErrorView } from "./ErrorView";
import WorkflowGridSkeleton from "./WorkflowGridSkeleton";
import {
  fetchWorkflows,
  notifyWorkflowsChanged,
  WORKFLOWS_CHANGED_EVENT,
  WORKFLOWS_PAGE_SIZE,
} from "@/lib/workflows-api";
import { Workflow } from "@/lib/types";
import { useDebounce } from "@/hooks/debounceHook";
import { Input } from "@/components/ui/input";
import { Loader2, Search, X } from "lucide-react";
import React, {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useVirtualizer } from "@tanstack/react-virtual";

function useColumnCount() {
  const [cols, setCols] = useState(1);

  useEffect(() => {
    const compute = () => {
      const w = window.innerWidth;
      // Full-width grid: more columns as the viewport grows (not centered/max-width capped).
      if (w >= 1920) setCols(7);
      else if (w >= 1536) setCols(6);
      else if (w >= 1280) setCols(5);
      else if (w >= 1024) setCols(4);
      else if (w >= 768) setCols(3);
      else if (w >= 640) setCols(2);
      else setCols(1);
    };
    compute();
    window.addEventListener("resize", compute);
    return () => window.removeEventListener("resize", compute);
  }, []);

  return cols;
}

function getScrollParent(el: HTMLElement | null): HTMLElement | null {
  if (!el) return null;
  let node: HTMLElement | null = el.parentElement;
  while (node) {
    const { overflowY } = window.getComputedStyle(node);
    if (overflowY === "auto" || overflowY === "scroll" || overflowY === "overlay") {
      return node;
    }
    node = node.parentElement;
  }
  return null;
}

export default function WorkflowGrid() {
  const [searchInput, setSearchInput] = useState("");
  const debouncedSearch = useDebounce(searchInput, 350);
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [page, setPage] = useState(1);
  const [hasNextPage, setHasNextPage] = useState(false);
  const [total, setTotal] = useState(0);
  const [initialLoading, setInitialLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [reloadToken, setReloadToken] = useState(0);

  const listRef = useRef<HTMLDivElement>(null);
  const loadMoreRef = useRef<HTMLDivElement>(null);
  const scrollParentRef = useRef<HTMLElement | null>(null);
  const columnCount = useColumnCount();
  const requestIdRef = useRef(0);

  const loadPage = useCallback(
    async (pageToLoad: number, search: string, append: boolean) => {
      const reqId = ++requestIdRef.current;
      if (append) {
        setLoadingMore(true);
      } else {
        setInitialLoading(true);
        setError(null);
      }

      const result = await fetchWorkflows({
        page: pageToLoad,
        limit: WORKFLOWS_PAGE_SIZE,
        search,
        sortOrder: "desc",
      });

      if (reqId !== requestIdRef.current) return;

      if (result.error) {
        setError(result.error);
        if (!append) {
          setWorkflows([]);
          setHasNextPage(false);
          setTotal(0);
        }
      } else {
        setError(null);
        setWorkflows((prev) =>
          append ? [...prev, ...result.workflows] : result.workflows
        );
        setPage(result.metadata.page);
        setHasNextPage(Boolean(result.metadata.hasNextPage));
        setTotal(result.metadata.total);
      }

      setInitialLoading(false);
      setLoadingMore(false);
    },
    []
  );

  useEffect(() => {
    setWorkflows([]);
    setPage(1);
    setHasNextPage(false);
    void loadPage(1, String(debouncedSearch ?? ""), false);
  }, [debouncedSearch, reloadToken, loadPage]);

  useEffect(() => {
    const onChanged = () => setReloadToken((t) => t + 1);
    window.addEventListener(WORKFLOWS_CHANGED_EVENT, onChanged);
    return () => window.removeEventListener(WORKFLOWS_CHANGED_EVENT, onChanged);
  }, []);

  const loadMore = useCallback(() => {
    if (!hasNextPage || loadingMore || initialLoading) return;
    void loadPage(page + 1, String(debouncedSearch ?? ""), true);
  }, [
    hasNextPage,
    loadingMore,
    initialLoading,
    loadPage,
    page,
    debouncedSearch,
  ]);

  useLayoutEffect(() => {
    scrollParentRef.current = getScrollParent(listRef.current);
  }, [initialLoading, workflows.length]);

  useEffect(() => {
    const target = loadMoreRef.current;
    if (!target) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) loadMore();
      },
      {
        root: scrollParentRef.current,
        rootMargin: "320px",
        threshold: 0,
      }
    );
    observer.observe(target);
    return () => observer.disconnect();
  }, [loadMore, workflows.length, hasNextPage]);

  const rows = useMemo(() => {
    const out: Workflow[][] = [];
    for (let i = 0; i < workflows.length; i += columnCount) {
      out.push(workflows.slice(i, i + columnCount));
    }
    return out;
  }, [workflows, columnCount]);

  const [scrollMargin, setScrollMargin] = useState(0);
  useLayoutEffect(() => {
    const list = listRef.current;
    const scrollParent = scrollParentRef.current ?? getScrollParent(list);
    scrollParentRef.current = scrollParent;
    if (!list || !scrollParent) {
      setScrollMargin(0);
      return;
    }
    const listTop = list.getBoundingClientRect().top;
    const parentTop = scrollParent.getBoundingClientRect().top;
    setScrollMargin(listTop - parentTop + scrollParent.scrollTop);
  }, [workflows.length, initialLoading, columnCount]);

  const rowVirtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () =>
      scrollParentRef.current ?? getScrollParent(listRef.current),
    estimateSize: () => 220,
    overscan: 4,
    scrollMargin,
  });

  useLayoutEffect(() => {
    rowVirtualizer.measure();
  }, [columnCount, rowVirtualizer, scrollMargin]);

  const onDeleted = useCallback((id: string) => {
    setWorkflows((prev) => prev.filter((w) => w.id !== id));
    setTotal((t) => Math.max(0, t - 1));
    notifyWorkflowsChanged();
  }, []);

  const onUpdated = useCallback((workflow: Workflow) => {
    setWorkflows((prev) =>
      prev.map((w) => (w.id === workflow.id ? { ...w, ...workflow } : w))
    );
  }, []);

  return (
    <div className="flex flex-col gap-4 w-full">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 w-full">
        <div className="relative w-full sm:max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
          <Input
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            placeholder="Search workflows by name or description…"
            className="pl-9 pr-9"
            aria-label="Search workflows"
          />
          {searchInput ? (
            <button
              type="button"
              aria-label="Clear search"
              className="absolute right-2 top-1/2 -translate-y-1/2 p-1 rounded-md text-muted-foreground hover:text-foreground"
              onClick={() => setSearchInput("")}
            >
              <X className="h-4 w-4" />
            </button>
          ) : null}
        </div>
        <p className="text-xs sm:text-sm text-muted-foreground sm:text-right">
          {initialLoading
            ? "Loading…"
            : `${total} workflow${total === 1 ? "" : "s"}`}
          {debouncedSearch ? ` matching “${debouncedSearch}”` : ""}
        </p>
      </div>

      {error && !workflows.length ? (
        <ErrorView
          error={error}
          onRetry={() => setReloadToken((t) => t + 1)}
        />
      ) : initialLoading ? (
        <WorkflowGridSkeleton />
      ) : !workflows.length ? (
        <p className="text-center p-8 text-muted-foreground">
          {debouncedSearch
            ? "No workflows match your search."
            : "No workflows found."}
        </p>
      ) : (
        <>
          <div ref={listRef} className="w-full">
            <div
              style={{
                height: `${rowVirtualizer.getTotalSize()}px`,
                width: "100%",
                position: "relative",
              }}
            >
              {rowVirtualizer.getVirtualItems().map((virtualRow) => {
                const row = rows[virtualRow.index] ?? [];
                return (
                  <div
                    key={virtualRow.key}
                    data-index={virtualRow.index}
                    ref={rowVirtualizer.measureElement}
                    className="absolute left-0 top-0 w-full"
                    style={{
                      transform: `translateY(${
                        virtualRow.start - scrollMargin
                      }px)`,
                    }}
                  >
                    <div
                      className="grid gap-4 pb-4"
                      style={{
                        gridTemplateColumns: `repeat(${columnCount}, minmax(0, 1fr))`,
                      }}
                    >
                      {row.map((workflow) => (
                        <WorkflowCard
                          key={workflow.id}
                          workflow={workflow}
                          onDeleted={onDeleted}
                          onUpdated={onUpdated}
                        />
                      ))}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          <div
            ref={loadMoreRef}
            className="flex items-center justify-center py-2 min-h-6"
            aria-hidden={!loadingMore}
          >
            {loadingMore ? (
              <span className="inline-flex items-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                Loading more…
              </span>
            ) : null}
          </div>
        </>
      )}
    </div>
  );
}
