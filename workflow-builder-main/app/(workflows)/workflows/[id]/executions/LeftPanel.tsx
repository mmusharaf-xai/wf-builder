"use client";
import { useWorkflowStore } from "@/app/store";
import LoadingSpinner from "@/components/loaders/SpinnerLoader";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DateTimeRangePicker,
  DateTimeRangeValue,
  rangeToQueryParams,
} from "@/components/ui/date-time-range-picker";
import {
  ChevronLeftSquareIcon,
  ChevronRightSquareIcon,
  RefreshCw,
} from "lucide-react";
import { useParams, useSearchParams } from "next/navigation";
import { WorkflowHistoryCard } from "./components/WorkflowHistoryCard";

import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useCallback, useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

function LeftPanel() {
  const searchParams = useSearchParams();
  const executionId = searchParams.get("e_id");
  const { id } = useParams();
  const {
    executionState: {
      current_page,
      total_pages,
      listLoading,
      listError,
      executions,
      page_size,
    },
    updateExecutionState,
  } = useWorkflowStore();

  const [dateRange, setDateRange] = useState<DateTimeRangeValue>({});
  const [refreshToken, setRefreshToken] = useState(0);

  const onChangePageSize = (value: string) => {
    updateExecutionState({ page_size: parseInt(value), current_page: 1 });
  };

  const onDateRangeChange = (value: DateTimeRangeValue) => {
    setDateRange(value);
    updateExecutionState({ current_page: 1 });
  };

  const getWorkflowHistory = useCallback(async () => {
    if (!id || isNaN(current_page)) {
      return;
    }
    updateExecutionState({ listLoading: true, listError: "" });
    try {
      const params = new URLSearchParams({
        page_number: String(current_page),
        page_size: String(page_size),
      });
      const { from, to } = rangeToQueryParams(dateRange);
      if (from) params.set("from", from);
      if (to) params.set("to", to);

      const response = await apiFetch(
        `/api/workflows/${id}/executions?${params.toString()}`
      );
      const responseData = await response.json();

      if (responseData?.error === false) {
        updateExecutionState({
          executions: responseData?.data?.result || [],
          total_pages: responseData?.data?.total_pages ?? 0,
          total_count: responseData?.data?.total_records ?? 0,
          listLoading: false,
        });
      } else if (responseData?.message) {
        updateExecutionState({
          listError: responseData?.message,
          executions: [],
          listLoading: false,
        });
      } else {
        throw new Error("Something went wrong");
      }
    } catch (error) {
      console.error("Error fetching workflow history:", error);
      updateExecutionState({
        listError: "Something went wrong.Please try again.",
        total_pages: 0,
        total_count: 0,
        executions: [],
        listLoading: false,
      });
    }
  }, [id, current_page, page_size, dateRange, updateExecutionState]);

  useEffect(() => {
    void getWorkflowHistory();
  }, [getWorkflowHistory, refreshToken]);

  const showPager = total_pages >= 1 || (executions?.length ?? 0) > 0;

  return (
    <div className="h-full min-h-0 w-full flex flex-col overflow-hidden">
      <header className="border-b p-2 sm:p-3 shrink-0 space-y-2">
        <DateTimeRangePicker
          value={dateRange}
          onChange={onDateRangeChange}
          placeholder="Filter by date & time"
          className="w-full"
        />

        {showPager ? (
          <div className="flex-wrap flex justify-center w-full items-center gap-1.5 sm:gap-2 text-xs">
            <Button
              disabled={current_page <= 1}
              onClick={() => {
                updateExecutionState({ current_page: current_page - 1 });
              }}
              variant={"outline"}
              size={"xs"}
              className="touch-manipulation"
            >
              <ChevronLeftSquareIcon />
            </Button>
            <Input
              className="max-w-[3rem] h-7"
              onChange={(e) => {
                const value = parseInt(e.target.value);
                if (value < 1 || value > total_pages) return;
                updateExecutionState({ current_page: value });
              }}
              value={current_page}
              type="number"
            />
            <div>/ {Math.max(total_pages, 1)}</div>
            <Button
              disabled={current_page >= total_pages || total_pages < 1}
              variant={"outline"}
              size={"xs"}
              className="touch-manipulation"
              onClick={() => {
                updateExecutionState({ current_page: current_page + 1 });
              }}
            >
              <ChevronRightSquareIcon />
            </Button>
            <Select value={`${page_size}`} onValueChange={onChangePageSize}>
              <SelectTrigger className="w-[65px] h-7">
                <SelectValue placeholder="page size" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="25">25</SelectItem>
                  <SelectItem value="50">50</SelectItem>
                  <SelectItem value="100">100</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
            <Button
              disabled={listLoading}
              onClick={() => setRefreshToken((t) => t + 1)}
              variant={"outline"}
              size={"xs"}
              className="touch-manipulation"
            >
              <RefreshCw />
            </Button>
          </div>
        ) : (
          <div className="flex justify-end">
            <Button
              disabled={listLoading}
              onClick={() => setRefreshToken((t) => t + 1)}
              variant={"outline"}
              size={"xs"}
              className="touch-manipulation"
            >
              <RefreshCw />
            </Button>
          </div>
        )}
      </header>

      <main className="flex-1 min-h-0 w-full pb-4 flex flex-col divide-y divide-neutral-200 gap-1 overflow-y-auto overscroll-contain relative p-2">
        <LoadingSpinner isLoading={listLoading} />
        {listError ? (
          <h1 className="h-full text-center w-full flex items-center justify-center p-4 text-sm">
            {listError}
          </h1>
        ) : executions?.length ? (
          executions.map((each, i) => {
            return (
              <WorkflowHistoryCard
                key={`${each.id} - ${i}`}
                isActive={each.id === executionId}
                item={each}
              />
            );
          })
        ) : (
          <div className="h-full w-full flex items-center font-semibold justify-center text-sm p-4 text-center">
            {dateRange.from || dateRange.to
              ? "No executions in this date range"
              : "No Executions Found"}
          </div>
        )}
      </main>
    </div>
  );
}

export default LeftPanel;
