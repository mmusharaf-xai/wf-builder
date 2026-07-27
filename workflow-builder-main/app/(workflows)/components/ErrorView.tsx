"use client";
import { Button } from "@/components/ui/button";
import { useRouter } from "next/navigation";

export const ErrorView = ({
  error,
  onRetry,
}: {
  error: string;
  onRetry?: () => void;
}) => {
  const router = useRouter();
  return (
    <div className="text-center p-4">
      <p className="text-red-500 mb-4">Error: {error}</p>
      <Button
        onClick={() => {
          if (onRetry) onRetry();
          else router.refresh();
        }}
      >
        Retry
      </Button>
    </div>
  );
};
