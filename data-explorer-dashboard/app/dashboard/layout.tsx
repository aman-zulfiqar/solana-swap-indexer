"use client"

import type React from "react"

import {
  SidebarProvider,
  Sidebar,
  SidebarContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
  SidebarFooter,
} from "@/components/ui/sidebar"
import { Database, MessageSquare, Flag, Circle } from "lucide-react"
import Link from "next/link"
import { usePathname } from "next/navigation"
import { useEffect, useState } from "react"

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8090/v1"

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const pathname = usePathname()
  const [healthStatus, setHealthStatus] = useState<boolean | null>(null)

  useEffect(() => {
    const checkHealth = async () => {
      try {
        const response = await fetch(`${API_BASE_URL}/health`)
        const data = await response.json()
        setHealthStatus(data.ok)
      } catch (error) {
        setHealthStatus(false)
      }
    }

    checkHealth()
    const interval = setInterval(checkHealth, 30000) // Check every 30s

    return () => clearInterval(interval)
  }, [])

  const menuItems = [
    {
      title: "Dashboard",
      href: "/dashboard",
      icon: Database,
    },
    {
      title: "AI Query",
      href: "/dashboard/ai",
      icon: MessageSquare,
    },
    {
      title: "Feature Flags",
      href: "/dashboard/flags",
      icon: Flag,
    },
  ]

  return (
    <SidebarProvider>
      <div className="flex min-h-screen w-full">
        <Sidebar>
          <SidebarHeader className="border-b border-sidebar-border px-6 py-4">
            <h2 className="text-lg font-semibold">Data Explorer</h2>
          </SidebarHeader>
          <SidebarContent>
            <SidebarMenu>
              {menuItems.map((item) => (
                <SidebarMenuItem key={item.href}>
                  <SidebarMenuButton asChild isActive={pathname === item.href}>
                    <Link href={item.href}>
                      <item.icon className="h-4 w-4" />
                      <span>{item.title}</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarContent>
          <SidebarFooter className="border-t border-sidebar-border px-6 py-4">
            <div className="flex items-center gap-2 text-sm">
              <Circle
                className={`h-2 w-2 fill-current ${
                  healthStatus === null
                    ? "text-muted-foreground"
                    : healthStatus
                      ? "text-emerald-500"
                      : "text-destructive"
                }`}
              />
              <span className="text-muted-foreground">
                {healthStatus === null ? "Checking..." : healthStatus ? "System Healthy" : "System Offline"}
              </span>
            </div>
          </SidebarFooter>
        </Sidebar>
        <main className="flex-1 p-8">
          <div className="mx-auto max-w-7xl">{children}</div>
        </main>
      </div>
    </SidebarProvider>
  )
}
