import type { Metadata, Viewport } from "next";
import { Analytics } from "@vercel/analytics/next";
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
      { url: "/icon-light-32x32.png", media: "(prefers-color-scheme: light)" },
      { url: "/icon-dark-32x32.png", media: "(prefers-color-scheme: dark)" },
      { url: "/icon.svg", type: "image/svg+xml" },
    ],
    apple: "/apple-icon.png",
  },
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
        <Analytics />
      </body>
    </html>
  );
}
