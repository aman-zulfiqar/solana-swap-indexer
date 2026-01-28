"use client";

import { useState } from "react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Loader2, ChevronDown, ChevronRight, Sparkles } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { askAI } from "@/lib/api";

interface AiResponse {
  sql: string;
  answer: string;
  took_ms: number;
}

export function AiQueryInterface() {
  const [question, setQuestion] = useState("");
  const [loading, setLoading] = useState(false);
  const [response, setResponse] = useState<AiResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [sqlExpanded, setSqlExpanded] = useState(false);

  const handleSubmit = async () => {
    if (!question.trim()) return;

    setLoading(true);
    setError(null);
    setResponse(null);

    try {
      const data = await askAI(question);
      setResponse(data);
    } catch (err) {
      if (err instanceof Error && err.message.includes("429")) {
        setError("Rate limit exceeded. Please try again in a moment.");
      } else {
        setError(
          err instanceof Error ? err.message : "Failed to process query"
        );
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      <Card className="p-6">
        <div className="space-y-4">
          <div>
            <label
              htmlFor="question"
              className="text-sm font-medium mb-2 block"
            >
              Ask a question about your data
            </label>
            <Textarea
              id="question"
              placeholder="e.g., What are the top 10 DEXs by trading volume today?"
              value={question}
              onChange={(e) => setQuestion(e.target.value)}
              rows={4}
              className="resize-none"
            />
          </div>
          <Button
            onClick={handleSubmit}
            disabled={loading || !question.trim()}
            className="w-full"
          >
            {loading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Thinking... (this may take up to 5 seconds)
              </>
            ) : (
              <>
                <Sparkles className="mr-2 h-4 w-4" />
                Ask AI
              </>
            )}
          </Button>
        </div>
      </Card>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {response && (
        <div className="space-y-4">
          <Card className="p-6">
            <h3 className="text-lg font-semibold mb-3">Answer</h3>

            {Array.isArray(response.answer) ? (
              <div className="overflow-x-auto">
                <table className="w-full table-auto border border-border rounded-md">
                  <thead className="bg-muted text-left">
                    <tr>
                      {Object.keys(response.answer[0]).map((key) => (
                        <th key={key} className="px-3 py-2 border-b">
                          {key}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {response.answer.map(
                      (row: Record<string, any>, idx: number) => (
                        <tr key={idx} className="hover:bg-muted/50">
                          {Object.values(row).map((val, i) => (
                            <td key={i} className="px-3 py-2 border-b">
                              {val}
                            </td>
                          ))}
                        </tr>
                      )
                    )}
                  </tbody>
                </table>
              </div>
            ) : (
              <p className="text-foreground leading-relaxed whitespace-pre-wrap">
                {response.answer}
              </p>
            )}

            <p className="text-muted-foreground text-sm mt-4">
              Query completed in {response.took_ms}ms
            </p>
          </Card>

          <Card className="p-6">
            <Collapsible open={sqlExpanded} onOpenChange={setSqlExpanded}>
              <CollapsibleTrigger className="flex items-center gap-2 font-semibold hover:text-primary transition-colors">
                {sqlExpanded ? (
                  <ChevronDown className="h-4 w-4" />
                ) : (
                  <ChevronRight className="h-4 w-4" />
                )}
                View Generated SQL
              </CollapsibleTrigger>
              <CollapsibleContent className="mt-4">
                <pre className="bg-muted p-4 rounded-lg overflow-x-auto">
                  <code className="text-sm font-mono">{response.sql}</code>
                </pre>
              </CollapsibleContent>
            </Collapsible>
          </Card>
        </div>
      )}
    </div>
  );
}
