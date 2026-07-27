import React from "react";

type Props = {
  children: React.ReactNode;
  className?:string
};

function GlobalLayout({ children ,className=''}: Props) {
  return (
    <div
      className={`${className} border-t md:border-l border-muted-foreground/20 p-3 sm:p-4 pb-24 md:pb-20 h-full min-h-0 w-full md:rounded-l-3xl overflow-auto`}
    >
      {children}
    </div>
  );
}

export default GlobalLayout;
