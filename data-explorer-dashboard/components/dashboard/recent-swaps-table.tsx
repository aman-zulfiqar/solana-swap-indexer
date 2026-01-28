"use client"

import { useEffect, useState } from "react"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Card } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { getRecentSwaps } from "@/lib/api"

interface SwapEvent {
  signature: string
  timestamp: string
  pair: string
  token_in: string
  token_out: string
  amount_in: number
  amount_out: number
  price: number
  dex: string
}

export function RecentSwapsTable() {
  const [swaps, setSwaps] = useState<SwapEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchSwaps = async () => {
      try {
        const data = await getRecentSwaps()
        setSwaps(data.items || [])
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load swaps")
      } finally {
        setLoading(false)
      }
    }

    fetchSwaps()
    const interval = setInterval(fetchSwaps, 10000)

    return () => clearInterval(interval)
  }, [])

  if (loading) {
    return (
      <Card className="p-6">
        <div className="space-y-3">
          {[...Array(10)].map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      </Card>
    )
  }

  if (error) {
    return (
      <Card className="p-6">
        <p className="text-destructive text-center">{error}</p>
      </Card>
    )
  }

  return (
    <Card>
      <div className="overflow-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[180px]">Time</TableHead>
              <TableHead>DEX</TableHead>
              <TableHead>Pair</TableHead>
              <TableHead className="text-right">Amount In</TableHead>
              <TableHead>Token In</TableHead>
              <TableHead className="text-right">Amount Out</TableHead>
              <TableHead>Token Out</TableHead>
              <TableHead className="text-right">Price</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {swaps.map((swap) => (
              <TableRow key={swap.signature}>
                <TableCell className="font-mono text-xs">{new Date(swap.timestamp).toLocaleString()}</TableCell>
                <TableCell>
                  <span className="rounded-full bg-accent px-2 py-1 text-xs font-medium">{swap.dex}</span>
                </TableCell>
                <TableCell className="font-medium">{swap.pair}</TableCell>
                <TableCell className="text-right font-mono">
                  {swap.amount_in.toLocaleString(undefined, {
                    maximumFractionDigits: 4,
                  })}
                </TableCell>
                <TableCell className="text-muted-foreground">{swap.token_in}</TableCell>
                <TableCell className="text-right font-mono">
                  {swap.amount_out.toLocaleString(undefined, {
                    maximumFractionDigits: 4,
                  })}
                </TableCell>
                <TableCell className="text-muted-foreground">{swap.token_out}</TableCell>
                <TableCell className="text-right font-mono">${swap.price.toFixed(4)}</TableCell>
              </TableRow>
            ))}
            {swaps.length === 0 && (
              <TableRow>
                <TableCell colSpan={8} className="text-center text-muted-foreground py-8">
                  No recent swaps available
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </Card>
  )
}
