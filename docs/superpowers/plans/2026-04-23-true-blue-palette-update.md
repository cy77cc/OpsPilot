# "True Blue" Color Palette Update Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Perform a comprehensive color palette update to match the "True Blue" tech aesthetic, ensuring high clarity and consistent blue accents across the project.

**Architecture:** Surgical updates to configuration and component-level styles.

**Tech Stack:** Tailwind CSS, Ant Design, React.

---

### Task 1: Update Tailwind Configuration

**Files:**
- Modify: `web/tailwind.config.js`

- [ ] **Step 1: Update `primary` and `gray` palettes in `web/tailwind.config.js`**

```javascript
// Change primary palette from Indigo-based to a clean Blue palette:
// 50: '#f0f7ff', 100: '#e0efff', 500: '#1890ff', 600: '#096dd9', 700: '#0050b3'
// Refine gray palette to be more neutral.
```

- [ ] **Step 2: Commit changes**

```bash
git add web/tailwind.config.js
git commit -m "style: update tailwind primary and gray palettes to true blue"
```

---

### Task 2: Update Global CSS

**Files:**
- Modify: `web/src/index.css`

- [ ] **Step 1: Update body background and CSS variables**

```css
/* Change body background to #f0f2f5 */
/* Update --color-theme to #1890ff */
/* Update other related variables */
```

- [ ] **Step 2: Refine .ant-card styles**

```css
/* Adjust .ant-card border and shadow to be more subtle */
```

- [ ] **Step 3: Commit changes**

```bash
git add web/src/index.css
git commit -m "style: update global css variables and card styles"
```

---

### Task 3: Verify App ConfigProvider

**Files:**
- Modify: `web/src/App.tsx`

- [ ] **Step 1: Ensure `colorPrimary` is set to `#1890ff`**

- [ ] **Step 2: Commit changes (if any)**

```bash
git add web/src/App.tsx
git commit -m "style: ensure antd primary color is set to true blue"
```

---

### Task 4: Update KPIOverview Pastel Backgrounds

**Files:**
- Modify: `web/src/pages/Dashboard/components/KPIOverview.tsx`

- [ ] **Step 1: Update icon colors and backgrounds in `kpiData`**

```typescript
// Blue -> #e6f7ff, Green -> #f6ffed, Purple -> #f9f0ff, etc.
```

- [ ] **Step 2: Commit changes**

```bash
git add web/src/pages/Dashboard/components/KPIOverview.tsx
git commit -m "style: update kpi overview with professional pastel backgrounds"
```

---

### Task 5: Update Sidebar Active Styles

**Files:**
- Modify: `web/src/app/layout/AppLayout.tsx`

- [ ] **Step 1: Adjust sidebar active item styles**

```tsx
/* Ensure Sidebar active item background and text color use the new Blue */
```

- [ ] **Step 2: Commit changes**

```bash
git add web/src/app/layout/AppLayout.tsx
git commit -m "style: update sidebar active state colors"
```

---

### Task 6: Final Verification

- [ ] **Step 1: Run build to ensure no style regressions**

Run: `cd web && npm run build` (or similar)

- [ ] **Step 2: Verify visually (if possible, otherwise check generated CSS)**
