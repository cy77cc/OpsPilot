import React from 'react';
import type { PromptsItemType } from '@ant-design/x';

export interface CopilotSessionState {
  drawerWidth: number;
  inputValue: string;
  promptItems: PromptsItemType[];
  isBootstrapping: boolean;
  attachedFiles: File[];
}

type CopilotSessionAction =
  | { type: 'set_drawer_width'; width: number }
  | { type: 'set_input_value'; value: string }
  | { type: 'set_prompt_items'; items: PromptsItemType[] }
  | { type: 'set_bootstrapping'; value: boolean }
  | { type: 'append_attached_files'; files: File[] }
  | { type: 'remove_attached_file'; index: number }
  | { type: 'clear_attached_files' };

export function copilotSessionReducer(
  state: CopilotSessionState,
  action: CopilotSessionAction,
): CopilotSessionState {
  switch (action.type) {
    case 'set_drawer_width':
      return { ...state, drawerWidth: action.width };
    case 'set_input_value':
      return { ...state, inputValue: action.value };
    case 'set_prompt_items':
      return { ...state, promptItems: action.items };
    case 'set_bootstrapping':
      return { ...state, isBootstrapping: action.value };
    case 'append_attached_files':
      return { ...state, attachedFiles: [...state.attachedFiles, ...action.files] };
    case 'remove_attached_file':
      return {
        ...state,
        attachedFiles: state.attachedFiles.filter((_, index) => index !== action.index),
      };
    case 'clear_attached_files':
      return { ...state, attachedFiles: [] };
    default:
      return state;
  }
}

export function useCopilotSessionReducer(initialPromptItems: PromptsItemType[]) {
  const [state, dispatch] = React.useReducer(copilotSessionReducer, {
    drawerWidth: 736,
    inputValue: '',
    promptItems: initialPromptItems,
    isBootstrapping: false,
    attachedFiles: [],
  });

  return {
    state,
    setDrawerWidth: (width: number) => dispatch({ type: 'set_drawer_width', width }),
    setInputValue: (value: string) => dispatch({ type: 'set_input_value', value }),
    setPromptItems: (items: PromptsItemType[]) => dispatch({ type: 'set_prompt_items', items }),
    setIsBootstrapping: (value: boolean) => dispatch({ type: 'set_bootstrapping', value }),
    addAttachedFiles: (files: File[]) => dispatch({ type: 'append_attached_files', files }),
    removeAttachedFile: (index: number) => dispatch({ type: 'remove_attached_file', index }),
    clearAttachedFiles: () => dispatch({ type: 'clear_attached_files' }),
  };
}
