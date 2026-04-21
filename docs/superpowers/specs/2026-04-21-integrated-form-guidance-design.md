# Spec: Integrated AI Form Guidance & Popover

## 1. Overview
The current form guidance implementation (`GuidedFormItem`) causes significant layout shifts by rendering field hints in the "extra" area of Ant Design's `Form.Item` on focus. Additionally, the visual style (Emerald green) is inconsistent with the project's primary branding (Indigo). 

This design unifies "Field Guidance" and "AI Assistance" into a single, non-invasive Popover experience triggered by a subtle icon.

## 2. Success Criteria
- [ ] Zero layout shift when an input field gains or loses focus.
- [ ] Integrated UI for both static field guidance and interactive AI assistance.
- [ ] Visual consistency using the project's Indigo-500 theme.
- [ ] Subtle, non-distracting triggers (icons) that appear on focus.

## 3. Architecture & Components

### 3.1 `GuidedFormItem.tsx` (Update)
- Remove `mergedExtra` logic that renders `FieldGuideCard` in the `extra` prop.
- Update `isFocused` logic to only control the visibility of the trigger icon.
- Pass both `guide` and `aiAssist` data to `AIFormAssistantPopover`.

### 3.2 `AIFormAssistantPopover.tsx` (Update)
- Refactor the Popover content to include an optional "Field Guide" section at the top.
- Apply new visual styles:
    - Header with "填写指引" (Field Guide) title and icon.
    - Two-column layout for "填写建议" (What to enter) and "推荐示例" (Example).
    - Integrated AI input area below the guide.
- Use `indigo-500` (#6366f1) for all primary accents and icons.

### 3.3 `AIFieldWrapper` (Update)
- Ensure the trigger icon (Sparkles/Help) is positioned absolutely within the input container to avoid shifting text or layout.
- Add a subtle pulse animation to the AI icon to indicate availability.

## 4. User Interaction Flow
1. **Focus**: User clicks into an input field.
2. **Trigger Appearance**: A small Indigo icon (Sparkles if AI is enabled, Help if only Guide is enabled) appears on the right side of the input.
3. **Click**: User clicks the icon.
4. **Popover Open**: The integrated Popover opens, showing field instructions and the AI prompt box.
5. **Action**: User reads instructions or uses AI to generate content.
6. **Blur**: Icon disappears when focus leaves (optional, or remains visible if preferred).

## 5. Visual Specifications
- **Primary Color**: `#6366f1` (Indigo-500)
- **Backgrounds**: 
    - Popover Header: `#f8fafc` (Slate-50)
    - Code Snippets/Examples: `#eef2ff` (Indigo-50)
- **Border**: `#e2e8f0` (Slate-200)
- **Animation**: 
    - Icon: 2s pulse (opacity/scale).
    - Popover: Standard Ant Design slide/fade.

## 6. Implementation Plan (High Level)
1. Modify `types.ts` to support integrated props.
2. Update `FieldGuideCard.tsx` (or merge its logic into the Popover).
3. Update `AIFormAssistantPopover.tsx` with the new layout.
4. Update `GuidedFormItem.tsx` to handle the new trigger and layout logic.
5. Verify with existing tests and add new ones for the integrated experience.
