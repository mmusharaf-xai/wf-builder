"use client";

import * as React from "react";
import { format, setHours, setMinutes, startOfDay, endOfDay } from "date-fns";
import { CalendarIcon, X } from "lucide-react";
import type { DateRange } from "react-day-picker";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "@/components/ui/drawer";
import { useMediaQuery } from "@/hooks/useMediaQuery";

export type DateTimeRangeValue = {
  from?: Date;
  to?: Date;
};

type Props = {
  value: DateTimeRangeValue;
  onChange: (value: DateTimeRangeValue) => void;
  className?: string;
  placeholder?: string;
  /** Align popover (desktop). */
  align?: "start" | "center" | "end";
};

function timeString(d?: Date) {
  if (!d) return "00:00";
  return format(d, "HH:mm");
}

function applyTime(date: Date, time: string, endOf = false): Date {
  const [h, m] = time.split(":").map((n) => parseInt(n, 10));
  if (Number.isNaN(h) || Number.isNaN(m)) {
    return endOf ? endOfDay(date) : startOfDay(date);
  }
  return setMinutes(setHours(date, h), m);
}

function formatRangeLabel(value: DateTimeRangeValue) {
  if (!value.from && !value.to) return null;
  const from = value.from
    ? format(value.from, "MMM d, yyyy HH:mm")
    : "…";
  const to = value.to ? format(value.to, "MMM d, yyyy HH:mm") : "…";
  return `${from} – ${to}`;
}

function RangePanel({
  value,
  onChange,
  onDone,
}: {
  value: DateTimeRangeValue;
  onChange: (v: DateTimeRangeValue) => void;
  onDone?: () => void;
}) {
  const range: DateRange | undefined =
    value.from || value.to
      ? { from: value.from, to: value.to }
      : undefined;

  const fromTime = timeString(value.from);
  const toTime = timeString(value.to ?? value.from);

  return (
    <div className="flex flex-col gap-3">
      <Calendar
        initialFocus
        mode="range"
        defaultMonth={value.from ?? value.to ?? new Date()}
        selected={range}
        onSelect={(next) => {
          if (!next) {
            onChange({});
            return;
          }
          const from = next.from
            ? applyTime(next.from, value.from ? fromTime : "00:00")
            : undefined;
          const to = next.to
            ? applyTime(next.to, value.to ? toTime : "23:59", true)
            : next.from
              ? applyTime(next.from, value.to ? toTime : "23:59", true)
              : undefined;
          onChange({ from, to });
        }}
        numberOfMonths={1}
        className="mx-auto"
      />

      <div className="grid grid-cols-2 gap-3 px-1">
        <div className="space-y-1.5">
          <Label htmlFor="from-time" className="text-xs">
            From time
          </Label>
          <Input
            id="from-time"
            type="time"
            value={fromTime}
            disabled={!value.from}
            className="h-9"
            onChange={(e) => {
              if (!value.from) return;
              onChange({
                ...value,
                from: applyTime(value.from, e.target.value),
              });
            }}
          />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="to-time" className="text-xs">
            To time
          </Label>
          <Input
            id="to-time"
            type="time"
            value={toTime}
            disabled={!value.from && !value.to}
            className="h-9"
            onChange={(e) => {
              const base = value.to ?? value.from;
              if (!base) return;
              onChange({
                ...value,
                to: applyTime(base, e.target.value, true),
              });
            }}
          />
        </div>
      </div>

      <div className="flex items-center gap-2 px-1 pb-1">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="flex-1"
          onClick={() => onChange({})}
        >
          Clear
        </Button>
        {onDone ? (
          <Button type="button" size="sm" className="flex-1" onClick={onDone}>
            Apply
          </Button>
        ) : null}
      </div>
    </div>
  );
}

/**
 * Mobile-responsive datetime range filter (shadcn Calendar + Popover / Drawer).
 */
export function DateTimeRangePicker({
  value,
  onChange,
  className,
  placeholder = "Filter by date & time",
  align = "start",
}: Props) {
  const isDesktop = useMediaQuery("(min-width: 768px)");
  const [open, setOpen] = React.useState(false);
  const label = formatRangeLabel(value);
  const hasValue = Boolean(value.from || value.to);

  const trigger = (
    <Button
      type="button"
      variant="outline"
      size="sm"
      className={cn(
        "h-8 justify-start text-left font-normal touch-manipulation w-full min-w-0",
        !hasValue && "text-muted-foreground",
        className
      )}
    >
      <CalendarIcon className="mr-2 h-3.5 w-3.5 shrink-0" />
      <span className="truncate flex-1 text-xs sm:text-sm">
        {label ?? placeholder}
      </span>
      {hasValue ? (
        <span
          role="button"
          tabIndex={0}
          aria-label="Clear date filter"
          className="ml-1 rounded-sm p-0.5 hover:bg-muted shrink-0"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            onChange({});
          }}
          onKeyDown={(e) => {
            if (e.key === "Enter" || e.key === " ") {
              e.preventDefault();
              e.stopPropagation();
              onChange({});
            }
          }}
        >
          <X className="h-3.5 w-3.5" />
        </span>
      ) : null}
    </Button>
  );

  if (isDesktop) {
    return (
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>{trigger}</PopoverTrigger>
        <PopoverContent className="w-auto p-0" align={align}>
          <RangePanel value={value} onChange={onChange} />
        </PopoverContent>
      </Popover>
    );
  }

  return (
    <Drawer open={open} onOpenChange={setOpen}>
      <DrawerTrigger asChild>{trigger}</DrawerTrigger>
      <DrawerContent className="max-h-[90dvh]">
        <DrawerHeader className="text-left pb-2">
          <DrawerTitle className="text-base">Date & time range</DrawerTitle>
        </DrawerHeader>
        <div className="overflow-y-auto px-2 pb-2">
          <RangePanel
            value={value}
            onChange={onChange}
            onDone={() => setOpen(false)}
          />
        </div>
        <DrawerFooter className="pt-0">
          <DrawerClose asChild>
            <Button variant="outline" className="w-full">
              Close
            </Button>
          </DrawerClose>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}

/** Serialize range for API query (ISO UTC). */
export function rangeToQueryParams(value: DateTimeRangeValue): {
  from?: string;
  to?: string;
} {
  return {
    from: value.from ? value.from.toISOString() : undefined,
    to: value.to ? value.to.toISOString() : undefined,
  };
}
