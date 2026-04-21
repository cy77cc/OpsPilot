export type FormAssistFieldMeta = {
  key: string;
  label: string;
  purpose: string;
  rules?: string;
  placeholder?: string;
  currentValue?: string;
};

export type FormAssistConfig = {
  scene: string;
  fieldMeta: FormAssistFieldMeta;
  getFormContext?: () => Record<string, unknown>;
  disabled?: boolean;
};

export type FormAssistRequest = {
  scene: string;
  user_prompt: string;
  field_meta: {
    key: string;
    label: string;
    purpose: string;
    rules?: string;
    placeholder?: string;
    current_value?: string;
  };
  form_context: Record<string, unknown>;
};
