import { AiQueryInterface } from "@/components/dashboard/ai-query-interface"

export default function AiQueryPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-semibold tracking-tight">AI Query</h1>
        <p className="text-muted-foreground mt-1">Ask questions in natural language and get SQL-powered insights</p>
      </div>
      <AiQueryInterface />
    </div>
  )
}
