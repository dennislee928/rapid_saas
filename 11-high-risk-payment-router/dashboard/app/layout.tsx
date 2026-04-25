import "./styles.css";
import type { ReactNode } from "react";

export const metadata = {
  title: "RouteKit Dashboard",
  description: "Merchant payment routing dashboard stub"
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}

