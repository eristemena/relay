import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./src/app/**/*.{ts,tsx}",
    "./src/features/**/*.{ts,tsx}",
    "./src/shared/**/*.{ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        base: "var(--color-base)",
        surface: "var(--color-surface)",
        raised: "var(--color-raised)",
        border: "var(--color-border)",
        brand: "var(--color-brand)",
        "brand-mid": "var(--color-brand-mid)",
        "brand-dim": "var(--color-brand-dim)",
        text: "var(--color-text)",
        "text-muted": "var(--color-text-muted)",
        success: "var(--color-success)",
        error: "var(--color-error)",
      },
      fontFamily: {
        display: ["var(--font-urbanist)"],
        sans: ["var(--font-dm-sans)"],
        mono: ["var(--font-jetbrains-mono)"],
      },
      boxShadow: {
        thinking: "0 0 0 1px var(--color-brand-mid), 0 0 20px rgba(124,58,237,0.35)",
        complete: "0 0 0 1px var(--color-success), 0 0 12px rgba(52,211,153,0.2)",
        idle: "0 0 0 1px var(--color-border)",
        error: "0 0 0 1px var(--color-error)",
      },
      transitionTimingFunction: {
        relay: "cubic-bezier(0.16, 1, 0.3, 1)",
      },
    },
  },
  plugins: [],
};

export default config;
