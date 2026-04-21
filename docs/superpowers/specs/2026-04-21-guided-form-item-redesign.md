# Spec: Guided Form Item Redesign (Manual Trigger)

Improve the user experience and visual appeal of the `GuidedFormItem` by replacing the auto-expanding focus-based guide with a manual-trigger exclamation icon next to the field label.

## Context
The current `GuidedFormItem` automatically expands a large "Field Guide Card" below the input when it receives focus. This causes layout shifts, can be distracting for experienced users, and feels visually "heavy."

## Goals
- **Reduce UI Noise**: Only show guidance when explicitly requested.
- **Modernize Aesthetic**: Use a clean Popover-based UI instead of a static block.
- **Maintain Consistency**: Follow the project's existing color palette (Indigo/Slate) and component patterns.

## Design

### 1. Trigger Icon (The "!")
- **Position**: Placed immediately after the `Form.Item` label.
- **Visuals**: 
    - A small indigo exclamation circle icon (e.g., `InfoCircleOutlined` or a custom SVG).
    - Subtle hover effect (e.g., scale up or change opacity).
    - Size: ~14px to match label height.
- **Interaction**: Click (not hover) to toggle the guidance Popover.

### 2. Guidance Popover
- **Component**: Use Ant Design's `Popover`.
- **Placement**: `topRight` or `right` relative to the trigger icon.
- **Content (`FieldGuideCard`)**:
    - **Header**: Simple "Field Guide" or "Help" title with an icon.
    - **Body**: Grid or list of sections (Purpose, Example, etc.).
    - **Styling**:
        - Background: White or very light slate.
        - Borders: Soft shadows and subtle 1px border.
        - Typography: Use system font stack with 12px labels and 13px content.

### 3. Component Refactoring
- **`GuidedFormItem`**:
    - Remove focus/blur logic for visibility.
    - Update `Form.Item` to inject the icon into the `label` prop.
    - Ensure it handles both simple `string` labels and `ReactNode` labels.
- **`FieldGuideCard`**:
    - Refactor from a block-level container to a Popover-ready component.
    - Remove the emerald-green theme; replace with indigo/slate to align with the rest of the AI features.

## Testing Strategy
- **Visual Regression**: Verify the icon appears correctly next to different label lengths.
- **Interaction**: Ensure the Popover opens on click and closes on click-away/ESC.
- **Content**: Verify all guide sections (whatToEnter, purpose, example, etc.) render correctly when present.
- **Compatibility**: Ensure it doesn't break existing `Form.Item` features like validation messages or `extra` text.

## Alternatives Considered
- **Input Suffix**: Rejected because it might conflict with AI icons or password visibility toggles.
- **Hover Trigger**: Rejected to prevent accidental popups while moving the mouse.
