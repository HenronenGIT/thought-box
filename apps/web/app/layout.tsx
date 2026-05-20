import "./styles.css";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Thought Box",
  description: "Dictation-first thought capture",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}

