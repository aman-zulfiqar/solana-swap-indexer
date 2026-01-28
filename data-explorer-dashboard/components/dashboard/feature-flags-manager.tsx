"use client"

import { useEffect, useState } from "react"
import { Card } from "@/components/ui/card"
import { Switch } from "@/components/ui/switch"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { Plus, Trash2 } from "lucide-react"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { getFlagsList, createFlag, updateFlag, deleteFlag, type Flag } from "@/lib/api"

export function FeatureFlagsManager() {
  const [flags, setFlags] = useState<Flag[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [newFlagKey, setNewFlagKey] = useState("")
  const [newFlagValue, setNewFlagValue] = useState(false)

  const fetchFlags = async () => {
    try {
      const data = await getFlagsList()
      setFlags(data.items || [])
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load flags")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchFlags()
  }, [])

  const updateFlagHandler = async (key: string, value: boolean) => {
    try {
      await updateFlag(key, value)
      await fetchFlags()
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update flag")
    }
  }

  const addFlag = async () => {
    if (!newFlagKey.trim()) return

    try {
      await createFlag({ key: newFlagKey, value: newFlagValue })
      setNewFlagKey("")
      setNewFlagValue(false)
      await fetchFlags()
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to add flag")
    }
  }

  const deleteFlagHandler = async (key: string) => {
    try {
      await deleteFlag(key)
      await fetchFlags()
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete flag")
    }
  }


  if (loading) {
    return (
      <Card className="p-6">
        <div className="space-y-4">
          {[...Array(5)].map((_, i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Card className="p-6">
        <h3 className="text-lg font-semibold mb-4">Add New Flag</h3>
        <div className="flex gap-4 items-end">
          <div className="flex-1">
            <label htmlFor="newFlag" className="text-sm font-medium mb-2 block">
              Flag Key
            </label>
            <Input
              id="newFlag"
              placeholder="e.g., enable_new_feature"
              value={newFlagKey}
              onChange={(e) => setNewFlagKey(e.target.value)}
            />
          </div>
          <div className="flex items-center gap-2">
            <Switch checked={newFlagValue} onCheckedChange={setNewFlagValue} />
            <span className="text-sm text-muted-foreground">{newFlagValue ? "Enabled" : "Disabled"}</span>
          </div>
          <Button onClick={addFlag} disabled={!newFlagKey.trim()}>
            <Plus className="mr-2 h-4 w-4" />
            Add Flag
          </Button>
        </div>
      </Card>

      <Card>
        <div className="divide-y divide-border">
          {flags.map((flag) => (
            <div key={flag.key} className="p-6 flex items-center justify-between gap-4">
              <div className="flex-1">
                <h4 className="font-semibold font-mono">{flag.key}</h4>
                {/* removed: 'updated_at' property does not exist on type 'Flag' */}
                <div className="flex items-center gap-2">
                  <Switch checked={flag.value} onCheckedChange={(checked) => updateFlag(flag.key, checked)} />
                  <span className="text-sm font-medium min-w-[70px]">{flag.value ? "Enabled" : "Disabled"}</span>
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => deleteFlag(flag.key)}
                  className="text-destructive hover:text-destructive hover:bg-destructive/10"
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            </div>
          ))}
          {flags.length === 0 && (
            <div className="p-8 text-center text-muted-foreground">No feature flags configured yet</div>
          )}
        </div>
      </Card>
    </div>
  )
}
