import "./styles.css";

export const metadata = {
  title: "Aegis Trust Operations",
  description: "GateKeep compliance gateway and Reclaim anti-piracy operations dashboard"
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
