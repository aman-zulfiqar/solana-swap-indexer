import { RecentSwapsTable } from "@/components/dashboard/recent-swaps-table"

export default function DashboardPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-semibold tracking-tight">Recent Swaps</h1>
        <p className="text-muted-foreground mt-1">Real-time DEX swap activity across all pairs</p>
      </div>
      <RecentSwapsTable />
    </div>
  )
}
