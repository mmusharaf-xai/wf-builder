import type { Metadata } from "next";
import localFont from "next/font/local";
import "./globals.css";
import Sidebar from "@/components/ui/sidebar";
import { Toaster } from "@/components/ui/sonner";
import ModalProvider from "./providers/modalProvider";
import DrawerProvider from "./providers/drawerProvider";

const geistSans = localFont({
  src: "./fonts/GeistVF.woff",
  variable: "--font-geist-sans",
  weight: "100 900",
});
const geistMono = localFont({
  src: "./fonts/GeistMonoVF.woff",
  variable: "--font-geist-mono",
  weight: "100 900",
});

export const metadata: Metadata = {
  title: "Workflow Builder",
  description: "Build Automations Your Way.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        <div className="flex flex-col md:flex-row overflow-hidden h-[100dvh] h-screen">
          <DrawerProvider>
            <ModalProvider>
              <Sidebar />
              <Toaster />
              {/* pb for mobile bottom tab bar */}
              <div className="w-full flex-1 min-h-0 min-w-0 pb-16 md:pb-0 overflow-hidden">
                {children}
              </div>
            </ModalProvider>
          </DrawerProvider>
        </div>
      </body>
    </html>
  );
}
