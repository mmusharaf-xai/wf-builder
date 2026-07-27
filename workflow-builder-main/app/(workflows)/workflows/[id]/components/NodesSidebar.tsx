import {
  Card,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { nodesList } from "@/lib/constants";
import React, { useState } from "react";
import { onDragStart } from "../utils";
import NodeIconByType from "./IconsByNodeType";
import { useDrawer } from "@/app/providers/drawerProvider";
import { Button } from "@/components/ui/button";
import { TNodeTypes } from "@/lib/types";
import { Loader2 } from "lucide-react";

type NodesSidebarProps = {
  /** Called when the user clicks a node in the list (click-to-add). */
  onSelectNode?: (type: TNodeTypes) => void | Promise<void>;
};

function NodesSidebar({ onSelectNode }: NodesSidebarProps) {
  const { setClose } = useDrawer();
  const [pendingType, setPendingType] = useState<TNodeTypes | null>(null);
  // Suppress the click that browsers fire after a drag ends.
  const suppressClickUntil = React.useRef(0);

  const handleClick = async (type: TNodeTypes) => {
    if (Date.now() < suppressClickUntil.current) return;
    if (!onSelectNode || pendingType) return;
    setPendingType(type);
    try {
      await onSelectNode(type);
    } finally {
      setPendingType(null);
    }
  };

  return (
    <div className="h-full min-h-0 w-full flex flex-col gap-2 sm:gap-3">
      <p className="md:hidden text-xs text-muted-foreground px-0.5">
        Tap a node to add it to the canvas.
      </p>
      <div className="flex-1 min-h-0 flex flex-col gap-2 sm:gap-3 overflow-y-auto overscroll-contain pb-2">
        {nodesList.map((each) => {
          const isPending = pendingType === each.value;
          return (
            <Card
              key={each.value}
              // Drag-and-drop is desktop-oriented; mobile users tap to add.
              draggable={!pendingType}
              role="button"
              tabIndex={0}
              className="w-full cursor-pointer border rounded-lg bg-neutral-100 dark:border-neutral-700 dark:bg-neutral-900 hover:bg-neutral-200 dark:hover:bg-neutral-800 active:scale-[0.99] transition-colors touch-manipulation"
              onDragStart={(event) => {
                suppressClickUntil.current = Date.now() + 400;
                onDragStart(event, each.value);
              }}
              onDragEnd={() => {
                suppressClickUntil.current = Date.now() + 400;
              }}
              onClick={() => handleClick(each.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  void handleClick(each.value);
                }
              }}
            >
              <CardHeader className="flex flex-row items-center gap-3 sm:gap-4 p-3 sm:p-2 sm:px-4">
                {isPending ? (
                  <Loader2 className="h-5 w-5 animate-spin shrink-0" />
                ) : (
                  <div className="shrink-0">
                    <NodeIconByType type={each.value} />
                  </div>
                )}
                <CardTitle className="text-sm sm:text-md min-w-0">
                  {each.label}
                  <CardDescription className="text-xs line-clamp-2">
                    {each.description}
                  </CardDescription>
                </CardTitle>
              </CardHeader>
            </Card>
          );
        })}
      </div>
      <footer className="flex items-center border-t pt-3 mt-1 justify-end shrink-0 sticky bottom-0 bg-zinc-50 dark:bg-zinc-950">
        <Button onClick={setClose} className="w-full md:w-auto" variant="outline">
          Close
        </Button>
      </footer>
    </div>
  );
}

export default NodesSidebar;
