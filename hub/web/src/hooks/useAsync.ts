import { useState, useCallback } from 'react';

export interface UseAsyncOptions<T> {
  onSuccess?: (data: T) => void;
  onError?: (error: Error) => void;
}

export interface UseAsyncReturn<T, P extends unknown[]> {
  data: T | null;
  loading: boolean;
  error: Error | null;
  execute: (...args: P) => Promise<T | null>;
  reset: () => void;
}

export function useAsync<T, P extends unknown[] = []>(
  asyncFn: (...args: P) => Promise<T>,
  options?: UseAsyncOptions<T>
): UseAsyncReturn<T, P> {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const execute = useCallback(
    async (...args: P): Promise<T | null> => {
      setLoading(true);
      setError(null);
      try {
        const result = await asyncFn(...args);
        setData(result);
        options?.onSuccess?.(result);
        return result;
      } catch (err) {
        const error = err instanceof Error ? err : new Error(String(err));
        setError(error);
        options?.onError?.(error);
        return null;
      } finally {
        setLoading(false);
      }
    },
    [asyncFn, options]
  );

  const reset = useCallback(() => {
    setData(null);
    setError(null);
    setLoading(false);
  }, []);

  return { data, loading, error, execute, reset };
}

export default useAsync;
