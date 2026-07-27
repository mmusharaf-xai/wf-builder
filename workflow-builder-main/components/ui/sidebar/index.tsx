"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";
import React from "react";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { menuOptions } from "@/lib/constants";
import clsx from "clsx";
import { Separator } from "@/components/ui/separator";
import { Database, GitBranch, LucideMousePointerClick } from "lucide-react";

const MenuOptions = () => {
  const pathName = usePathname();

  return (
    <>
      {/* Desktop: left icon rail */}
      <nav className="hidden md:flex dark:bg-black h-screen overflow-auto justify-between items-center flex-col gap-10 py-6 px-2 shrink-0">
        <div className="flex items-center justify-center flex-col gap-8">
          <Link className="flex font-bold flex-row" href="/">
            Auto.
          </Link>
          <TooltipProvider>
            {menuOptions.map((menuItem) => (
              <ul key={menuItem.name}>
                <Tooltip delayDuration={0}>
                  <TooltipTrigger asChild>
                    <li>
                      <Link
                        href={menuItem.href}
                        className={clsx(
                          "group h-8 w-8 flex items-center justify-center scale-[1.5] rounded-lg p-[3px] cursor-pointer",
                          {
                            "dark:bg-[#2F006B] bg-[#EEE0FF]": pathName?.includes(
                              menuItem.href
                            ),
                          }
                        )}
                      >
                        <menuItem.Component
                          selected={pathName?.includes(menuItem.href)}
                        />
                      </Link>
                    </li>
                  </TooltipTrigger>
                  <TooltipContent
                    side="right"
                    className="bg-black/40 backdrop-blur-xl"
                  >
                    <p>{menuItem.name}</p>
                  </TooltipContent>
                </Tooltip>
              </ul>
            ))}
          </TooltipProvider>
          <Separator />
          <div className="flex items-center flex-col gap-9 dark:bg-[#353346]/30 py-4 px-2 rounded-full h-56 overflow-scroll border-[1px]">
            <div className="relative dark:bg-[#353346]/70 p-2 rounded-full dark:border-t-[2px] border-[1px] dark:border-t-[#353346]">
              <LucideMousePointerClick className="dark:text-white" size={18} />
              <div className="border-l-2 border-muted-foreground/50 h-6 absolute left-1/2 transform translate-x-[-50%] -bottom-[30px]" />
            </div>
            <div className="relative dark:bg-[#353346]/70 p-2 rounded-full dark:border-t-[2px] border-[1px] dark:border-t-[#353346]">
              <GitBranch className="text-muted-foreground" size={18} />
              <div className="border-l-2 border-muted-foreground/50 h-6 absolute left-1/2 transform translate-x-[-50%] -bottom-[30px]" />
            </div>
            <div className="relative dark:bg-[#353346]/70 p-2 rounded-full dark:border-t-[2px] border-[1px] dark:border-t-[#353346]">
              <Database className="text-muted-foreground" size={18} />
              <div className="border-l-2 border-muted-foreground/50 h-6 absolute left-1/2 transform translate-x-[-50%] -bottom-[30px]" />
            </div>
            <div className="relative dark:bg-[#353346]/70 p-2 rounded-full dark:border-t-[2px] border-[1px] dark:border-t-[#353346]">
              <GitBranch className="text-muted-foreground" size={18} />
            </div>
          </div>
        </div>
      </nav>

      {/* Mobile: bottom tab bar */}
      <nav
        className="md:hidden fixed bottom-0 inset-x-0 z-[100] border-t bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80 safe-area-pb"
        style={{ paddingBottom: "env(safe-area-inset-bottom, 0px)" }}
      >
        <ul className="flex items-stretch justify-around gap-1 px-1 py-1.5">
          {menuOptions.map((menuItem) => {
            const active = pathName?.includes(menuItem.href);
            return (
              <li key={menuItem.name} className="flex-1 min-w-0">
                <Link
                  href={menuItem.href}
                  className={clsx(
                    "flex flex-col items-center justify-center gap-0.5 rounded-md px-1 py-1.5 text-[10px] leading-tight",
                    active
                      ? "text-foreground bg-muted font-medium"
                      : "text-muted-foreground"
                  )}
                >
                  <span
                    className={clsx(
                      "flex h-8 w-8 items-center justify-center rounded-lg",
                      active && "dark:bg-[#2F006B] bg-[#EEE0FF]"
                    )}
                  >
                    <menuItem.Component selected={!!active} />
                  </span>
                  <span className="truncate max-w-full">{menuItem.name}</span>
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>
    </>
  );
};

export default MenuOptions;
