import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Consize Control Plane",
  description: "Policy-governed infrastructure optimization dashboard"
};

export default function RootLayout({
  children
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
