import { FeatureFlagsManager } from "@/components/dashboard/feature-flags-manager"

export default function FeatureFlagsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-semibold tracking-tight">Feature Flags</h1>
        <p className="text-muted-foreground mt-1">Manage feature toggles and configuration flags</p>
      </div>
      <FeatureFlagsManager />
    </div>
  )
}
