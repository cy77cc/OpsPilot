import { useState, useEffect, useCallback, useRef } from 'react';
import { aiApi } from '../../../api/modules/ai';
import type { FormAssistConfig } from '../types/formAssist';

/**
 * useFormAssist hook manages the state and logic for AI-assisted form fields.
 * It handles the global feature flag, hint display timer, and streaming suggestion logic.
 */
export function useFormAssist(
  config: FormAssistConfig | undefined,
  currentFieldValue: string,
  onApply: (value: string) => void
) {
  const [isOpen, setIsOpen] = useState(false);
  const [isStreaming, setIsStreaming] = useState(false);
  const [prompt, setPrompt] = useState('');
  const [preview, setPreview] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [showHint, setShowHint] = useState(false);

  const abortControllerRef = useRef<AbortController | null>(null);

  // Read the global feature flag from localStorage. Default to enabled ('1').
  const isEnabled = typeof window !== 'undefined' 
    ? (localStorage.getItem('ai-form-assist-enabled') !== '0') // Anything other than '0' (including null) is enabled
    : false;

  useEffect(() => {
    // Start a 3-second timer only when:
    // - the feature is enabled (flag == '1'),
    // - the field is opted in (config is provided),
    // - the field currently has a non-empty value (passed as currentFieldValue),
    // - the popover is closed,
    // - no stream request is in progress.
    if (!isEnabled || !config || !currentFieldValue || isOpen || isStreaming || config.disabled) {
      setShowHint(false);
      return;
    }

    const timer = setTimeout(() => {
      setShowHint(true);
    }, 3000);

    return () => clearTimeout(timer);
  }, [isEnabled, config, currentFieldValue, isOpen, isStreaming]);

  const open = useCallback(() => {
    setIsOpen(true);
    setShowHint(false);
  }, []);

  const close = useCallback(() => {
    setIsOpen(false);
  }, []);

  const cancel = useCallback(() => {
    setPrompt('');
    setPreview('');
    setError(null);
    setIsOpen(false);
    setIsStreaming(false);
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
  }, []);

  const submit = useCallback(
    async (userPrompt: string) => {
      if (!config) return;

      setPrompt(userPrompt);
      setPreview('');
      setError(null);
      setIsStreaming(true);

      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
      abortControllerRef.current = new AbortController();

      try {
        await aiApi.formAssistStream(
          {
            scene: config.scene,
            user_prompt: userPrompt,
            field_meta: {
              key: config.fieldMeta.key,
              label: config.fieldMeta.label,
              purpose: config.fieldMeta.purpose,
              rules: config.fieldMeta.rules,
              placeholder: config.fieldMeta.placeholder,
              current_value: currentFieldValue,
            },
            form_context: config.getFormContext?.() || {},
          },
          {
            onDelta: (delta) => {
              setPreview((prev) => prev + delta.content);
            },
            onError: (err) => {
              setError(err.message || 'AI streaming error');
              setIsStreaming(false);
            },
            onDone: () => {
              setIsStreaming(false);
            },
          },
          abortControllerRef.current.signal
        );
      } catch (err: any) {
        if (err.name === 'AbortError') {
          // Ignore abort errors
          return;
        }
        setError(err.message || 'Failed to connect to AI service');
        setIsStreaming(false);
      }
    },
    [config, currentFieldValue]
  );

  const applySuggestion = useCallback(() => {
    onApply(preview);
    close();
  }, [onApply, preview, close]);

  const dismissHint = useCallback(() => {
    setShowHint(false);
  }, []);

  return {
    isEnabled,
    isOpen,
    isStreaming,
    prompt,
    preview,
    error,
    showHint,
    open,
    close,
    cancel,
    submit,
    applySuggestion,
    dismissHint,
  };
}
