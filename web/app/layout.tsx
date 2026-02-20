import type { Metadata, Viewport } from "next";
import { TelegramProvider } from "@/context/TelegramProvider";
import { SocketProvider } from "@/context/SocketContext";
import { GameProvider } from "@/context/GameContext";
import "./globals.css";

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  maximumScale: 1,
  userScalable: false,
  viewportFit: "cover",
};

export const metadata: Metadata = {
  title: "Last Click",
  description: "High-fidelity volatility survival game",
  icons: {
    icon: [
      { url: "/favicon-32x32.png", sizes: "32x32", type: "image/png" },
      { url: "/favicon-16x16.png", sizes: "16x16", type: "image/png" },
      { url: "/favicon.ico", sizes: "any" },
    ],
    apple: "/apple-touch-icon.png",
  },
  manifest: "/site.webmanifest",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en" className="dark" suppressHydrationWarning>
      <body
        className="font-sans antialiased bg-background text-foreground overscroll-none"
        suppressHydrationWarning
      >
        <TelegramProvider>
          <SocketProvider>
            <GameProvider>{children}</GameProvider>
          </SocketProvider>
        </TelegramProvider>
      </body>
    </html>
  );
}
