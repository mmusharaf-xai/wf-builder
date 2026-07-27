"use client";
import * as React from "react";
import { Button } from "@/components/ui/button";
import { Drawer as VaulDrawer } from "vaul";
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { useDrawer } from "@/app/providers/drawerProvider";
import { usePathname } from "next/navigation";
import { useEffect } from "react";
import clsx from "clsx";

type Props = {
  title?: string;
  subheading?: string;
  children: React.ReactNode;
  defaultOpen?: boolean;
  modal?: boolean;
  hideHeader?: boolean;
  /** Hide the mobile footer Cancel button (when children already have one). */
  hideFooter?: boolean;
};

export default function DrawerComponent({
  children,
  defaultOpen,
  title,
  subheading,
  hideHeader = false,
  hideFooter = false,
  modal = false,
}: Props) {
  const { isOpen, setClose, isFullScreen } = useDrawer();
  const isDesktop = useMediaQuery("(min-width: 768px)");
  const pathname = usePathname();
  const handleClose = () => setClose();

  useEffect(() => {
    if (isOpen) {
      handleClose();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pathname]);

  useEffect(() => {
    window.requestAnimationFrame(() => {
      document.body.style.pointerEvents = "auto";
    });
  }, [isOpen]);

  if (isDesktop) {
    return (
      <VaulDrawer.Root
        direction={"right"}
        modal={modal}
        dismissible={false}
        defaultOpen={defaultOpen}
        open={isOpen}
      >
        <VaulDrawer.Portal>
          <VaulDrawer.Overlay className="fixed inset-0 bg-black/40 z-[998]" />
          <VaulDrawer.Content
            className={clsx(
              "fixed z-[999] outline-none flex transition-all duration-300",
              isFullScreen
                ? "inset-2"
                : "right-2 top-2 bottom-2 w-[min(100vw-1rem,360px)] max-w-[360px]"
            )}
            style={
              !isFullScreen
                ? ({
                    "--initial-transform": "calc(100% + 18px)",
                  } as React.CSSProperties)
                : undefined
            }
          >
            <div className="bg-zinc-50 dark:bg-zinc-950 h-full w-full grow p-4 sm:p-5 flex overflow-hidden flex-col rounded-[16px]">
              <VaulDrawer.Title
                hidden={hideHeader}
                className="font-medium text-zinc-900 dark:text-zinc-50 shrink-0"
              >
                {title}
              </VaulDrawer.Title>

              <VaulDrawer.Description
                hidden={hideHeader}
                className="text-zinc-600 dark:text-zinc-400 text-sm shrink-0"
              >
                {subheading}
              </VaulDrawer.Description>

              <div
                className={clsx(
                  "flex-1 min-h-0 overflow-auto w-full",
                  !hideHeader && (title || subheading) ? "mt-2" : "-mt-2"
                )}
              >
                {children}
              </div>
            </div>
          </VaulDrawer.Content>
        </VaulDrawer.Portal>
      </VaulDrawer.Root>
    );
  }

  // Mobile: bottom sheet with capped height + scrollable body
  return (
    <Drawer open={isOpen} onClose={handleClose} defaultOpen={defaultOpen}>
      <DrawerContent
        className={clsx(
          "max-h-[min(92dvh,92vh)] flex flex-col p-0",
          isFullScreen && "h-[min(95dvh,95vh)] max-h-[min(95dvh,95vh)]"
        )}
      >
        {!hideHeader && (title || subheading) && (
          <DrawerHeader className="text-left px-4 pt-2 pb-2 shrink-0 border-b">
            <div className="flex justify-between items-start gap-2">
              <div className="min-w-0">
                {title && (
                  <DrawerTitle className="text-base sm:text-lg truncate">
                    {title}
                  </DrawerTitle>
                )}
                {subheading && (
                  <DrawerDescription className="text-xs sm:text-sm line-clamp-2">
                    {subheading}
                  </DrawerDescription>
                )}
              </div>
            </div>
          </DrawerHeader>
        )}
        <div
          className={clsx(
            "flex-1 min-h-0 overflow-y-auto overscroll-contain px-4",
            !hideHeader && (title || subheading) ? "py-3" : "mt-2 py-3",
            // Leave room for bottom nav + footer
            hideFooter ? "pb-20" : "pb-2"
          )}
        >
          {children}
        </div>
        {!hideFooter && (
          <DrawerFooter
            className="pt-2 pb-4 px-4 shrink-0 border-t"
            style={{ paddingBottom: "max(1rem, env(safe-area-inset-bottom))" }}
          >
            <DrawerClose asChild>
              <Button variant="outline" className="w-full">
                Cancel
              </Button>
            </DrawerClose>
          </DrawerFooter>
        )}
      </DrawerContent>
    </Drawer>
  );
}
