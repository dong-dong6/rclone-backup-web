import { useState, useCallback } from 'react';

export interface UseClipboardReturn {
  copied: boolean;
  copy: (text: string) => Promise<void>;
  reset: () => void;
}

export function useClipboard(resetDelay = 2000): UseClipboardReturn {
  const [copied, setCopied] = useState(false);

  const copy = useCallback(
    async (text: string) => {
      try {
        if (navigator.clipboard && window.isSecureContext) {
          await navigator.clipboard.writeText(text);
        } else {
          // Fallback for older browsers
          const textArea = document.createElement('textarea');
          textArea.value = text;
          textArea.style.position = 'fixed';
          textArea.style.left = '-999999px';
          document.body.appendChild(textArea);
          textArea.select();
          document.execCommand('copy');
          document.body.removeChild(textArea);
        }
        setCopied(true);
        if (resetDelay > 0) {
          setTimeout(() => setCopied(false), resetDelay);
        }
      } catch (err) {
        console.error('Failed to copy:', err);
      }
    },
    [resetDelay]
  );

  const reset = useCallback(() => {
    setCopied(false);
  }, []);

  return { copied, copy, reset };
}

export default useClipboard;
