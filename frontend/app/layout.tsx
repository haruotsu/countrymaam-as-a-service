import type { Metadata } from "next";
import "./globals.css";
import { AuthProvider } from "@/lib/auth";
import { HeaderBar } from "./components/HeaderBar";

export const metadata: Metadata = {
  title: "Countrymaam as a Service",
  description: "カントリーマアムを預けて、送って、両替する銀行（パロディ）",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="ja">
      <body>
        <AuthProvider>
          <HeaderBar />
          <main>{children}</main>
        </AuthProvider>
      </body>
    </html>
  );
}
