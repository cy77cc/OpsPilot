# Integrated Form Guidance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate layout shifts in forms by integrating field guidance into a non-invasive AI-assisted Popover triggered by focus-visible icons.

**Architecture:** 
- Unify `FieldGuide` and `AIFormAssistant` into a single Popover content structure.
- Use absolute-positioned trigger icons within the input field to prevent layout displacement.
- Standardize on the `indigo-500` visual theme.

**Tech Stack:** React, Ant Design (Form, Popover, Tooltip), Tailwind CSS.

---

### Task 1: Update Types and Icons

**Files:**
- Modify: `web/src/components/FormGuidance/types.ts`
- Modify: `web/src/components/FormGuidance/AIFormAssistantPopover.tsx`

- [ ] **Step 1: Update `AIFormAssistantPopoverProps` to include guidance data**

Modify `web/src/components/FormGuidance/AIFormAssistantPopover.tsx`:
```typescript
// Add FieldGuide to imports
import type { FieldGuide } from './types';

// Update props interface
export interface AIFormAssistantPopoverProps {
  guide?: FieldGuide; // Add this
  isOpen: boolean;
  // ... existing props
}
```

- [ ] **Step 2: Commit**

```bash
git add web/src/components/FormGuidance/types.ts web/src/components/FormGuidance/AIFormAssistantPopover.tsx
git commit -m "chore: update guidance types and popover props"
```

---

### Task 2: Refactor `AIFormAssistantPopover` UI

**Files:**
- Modify: `web/src/components/FormGuidance/AIFormAssistantPopover.tsx`

- [ ] **Step 1: Implement the Field Guide section in the Popover**

Refactor the content of `AIFormAssistantPopover.tsx` to include the "填写指引" section at the top if `guide` is provided. Use the Indigo theme.

```tsx
const PopoverContent = (
  <div className="w-80 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xl">
    {guide && (
      <div className="bg-slate-50 p-4 border-bottom border-slate-100">
        <div className="flex items-center gap-2 mb-3">
          <div className="bg-indigo-50 text-indigo-500 p-1 rounded-md">
             <HelpCircleIcon size={14} />
          </div>
          <span className="text-[10px] font-bold text-slate-500 uppercase tracking-widest">填写指引</span>
        </div>
        <div className="space-y-3">
          {guide.whatToEnter && (
            <div>
              <div className="text-[10px] font-bold text-slate-400 uppercase mb-1">建议</div>
              <div className="text-sm text-slate-600 leading-relaxed">{guide.whatToEnter}</div>
            </div>
          )}
          {guide.example && (
            <div>
              <div className="text-[10px] font-bold text-slate-400 uppercase mb-1">示例</div>
              <code className="text-xs text-indigo-600 bg-indigo-50 px-1.5 py-0.5 rounded font-mono">
                {guide.example}
              </code>
            </div>
          )}
        </div>
      </div>
    )}
    
    {/* Existing AI Section */}
    <div className="p-4">
       {/* ... existing AI assistant UI ... */}
    </div>
  </div>
);
```

- [ ] **Step 2: Commit**

```bash
git add web/src/components/FormGuidance/AIFormAssistantPopover.tsx
git commit -m "feat: integrated field guidance into AI popover UI"
```

---

### Task 3: Update `GuidedFormItem` for Non-Invasive Triggers

**Files:**
- Modify: `web/src/components/FormGuidance/GuidedFormItem.tsx`

- [ ] **Step 1: Remove layout-shifting `mergedExtra` logic**

In `GuidedFormItem.tsx`, stop rendering `FieldGuideCard` in the `extra` prop.

```tsx
// BEFORE
const mergedExtra = isFocused && guide ? ( ... ) : extra;

// AFTER
const mergedExtra = extra;
```

- [ ] **Step 2: Update the trigger icon logic**

Ensure the icon (Sparkles or Help) only appears when focused, and uses the absolute-positioned `AIFieldWrapper`.

```tsx
const aiTrigger = (isFocused && (effectiveAiAssist || guide)) ? (
  <AIFormAssistantPopover
    guide={guide}
    isOpen={isOpen}
    // ...
  >
    <div className="flex items-center justify-center h-8 w-8 rounded-lg hover:bg-indigo-50 transition-all cursor-pointer text-indigo-500">
       {effectiveAiAssist ? <SparklesIcon className="animate-pulse" /> : <HelpCircleIcon />}
    </div>
  </AIFormAssistantPopover>
) : null;
```

- [ ] **Step 3: Run existing tests**

Run: `npm test web/src/components/FormGuidance/GuidedFormItem.test.tsx`
Expected: PASS (may need minor updates to snapshots or focus expectations)

- [ ] **Step 4: Commit**

```bash
git add web/src/components/FormGuidance/GuidedFormItem.tsx
git commit -m "feat: use popover for guidance to avoid layout shift"
```

---

### Task 4: Final Polish and Cleanup

**Files:**
- Delete: `web/src/components/FormGuidance/FieldGuideCard.tsx` (if no longer used)
- Modify: `web/src/components/FormGuidance/index.ts`

- [ ] **Step 1: Delete unused component**

```bash
rm web/src/components/FormGuidance/FieldGuideCard.tsx
```

- [ ] **Step 2: Verify overall form appearance**

Check that focus transitions are smooth and no vertical jumping occurs.

- [ ] **Step 3: Commit**

```bash
git add web/src/components/FormGuidance/
git commit -m "cleanup: remove unused FieldGuideCard component"
```
