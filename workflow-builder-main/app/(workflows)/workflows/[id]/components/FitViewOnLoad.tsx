"use client";

import { useEffect, useRef } from "react";
import { useNodesInitialized, useReactFlow, useStore } from "@xyflow/react";

type Props = {
  /** Unique id for the graph being shown (workflow id or execution id). */
  graphKey: string;
  /** When false, wait (e.g. still loading or panel hidden). */
  enabled?: boolean;
};

/**
 * Same as Controls "fit view", but after nodes + pane are ready.
 * Retries briefly so late layout (resizable panel / mobile tab) still fits.
 */
export default function FitViewOnLoad({ graphKey, enabled = true }: Props) {
  const { fitView } = useReactFlow();
  const nodesInitialized = useNodesInitialized({ includeHiddenNodes: false });
  const width = useStore((s) => s.width);
  const height = useStore((s) => s.height);
  const nodeCount = useStore((s) => s.nodes.length);
  const fittedKey = useRef<string | null>(null);

  useEffect(() => {
    fittedKey.current = null;
  }, [graphKey]);

  useEffect(() => {
    if (!enabled || !graphKey) return;
    if (!nodesInitialized || nodeCount === 0) return;
    if (!width || !height) return;
    if (fittedKey.current === graphKey) return;

    let cancelled = false;
    const delays = [0, 50, 100, 200, 400];
    const timers: number[] = [];

    const tryFit = async (attempt: number) => {
      if (cancelled || fittedKey.current === graphKey) return;
      try {
        // Same call as Controls fit button (no custom maxZoom/padding).
        const result = await fitView();
        // xyflow returns boolean | void depending on version
        if (result !== false) {
          fittedKey.current = graphKey;
          return;
        }
      } catch {
        // retry
      }
      if (attempt + 1 < delays.length && !cancelled) {
        timers.push(
          window.setTimeout(() => {
            void tryFit(attempt + 1);
          }, delays[attempt + 1] - delays[attempt])
        );
      }
    };

    timers.push(
      window.setTimeout(() => {
        void tryFit(0);
      }, delays[0])
    );

    return () => {
      cancelled = true;
      timers.forEach((t) => window.clearTimeout(t));
    };
  }, [
    enabled,
    graphKey,
    nodesInitialized,
    nodeCount,
    width,
    height,
    fitView,
  ]);

  return null;
}
