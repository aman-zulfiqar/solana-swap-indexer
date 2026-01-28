// app/page.tsx
import { redirect } from "next/navigation";

export default function Home() {
  // Server-side redirect
  redirect("/dashboard");
  return null; // never renders
}
