import { useState, useEffect } from "react"
import debounce from "debounce"

export function useDebouncedValue<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value)

  useEffect(() => {
    const setter = debounce(setDebounced, delay)
    setter(value)
    return () => { setter.clear() }
  }, [value, delay])

  return debounced
}
