import type { Metadata } from "next";
import "./globals.css";

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
        <header className="site">
          <h1>🍪 Countrymaam as a Service</h1>
          <p>
            カントリーマアムを資産として扱う、やや本気のパロディ銀行
          </p>
        </header>
        <main>{children}</main>
      </body>
    </html>
  );
}
