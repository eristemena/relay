import type { Metadata } from "next";
import { DM_Sans, JetBrains_Mono, Urbanist } from "next/font/google";
import "./globals.css";

const urbanist = Urbanist({
  subsets: ["latin"],
  variable: "--font-urbanist",
});

const dmSans = DM_Sans({
  subsets: ["latin"],
  variable: "--font-dm-sans",
});

const jetBrainsMono = JetBrains_Mono({
  subsets: ["latin"],
  variable: "--font-jetbrains-mono",
});

export const metadata: Metadata = {
  title: "Relay - Local Workspace",
  description: "Local browser workspace for Relay.",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body className={`${urbanist.variable} ${dmSans.variable} ${jetBrainsMono.variable} bg-base font-sans text-text`}>
        <a className="skip-link" href="#maincontent">
          Skip to main content
        </a>
        {children}
      </body>
    </html>
  );
}
