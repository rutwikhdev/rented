export function errorMessage(err: unknown, fallback = "an unexpected error occurred"): string {
  return err instanceof Error ? err.message : fallback
}
