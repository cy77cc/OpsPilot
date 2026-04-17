# Calm Emerald UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform OpsPilot's UI into the "Calm Emerald" style — featuring a nature-inspired palette (Forest Green/Warm Sand), high-density layout, and refined 12px geometry.

**Architecture:** We will adopt a top-down approach: first updating global design tokens (Tailwind & Ant Design), then refactoring the layout foundation, and finally migrating individual dashboard components to the new high-density standard.

**Tech Stack:** React, Tailwind CSS, Ant Design (v5), TypeScript.

---

### Task 1: Update Global Design Tokens

**Files:**
- Modify: `web/src/design-system/tokens.ts`
- Modify: `web/src/theme/antd-theme.ts`
- Modify: `web/tailwind.config.js`
- Modify: `web/src/index.css`

- [ ] **Step 1: Update design-system tokens**
Update the core tokens to reflect the "Calm Emerald" palette and 12px standard radius.

```typescript
// web/src/design-system/tokens.ts (partial update)
export const colors = {
  primary: {
    50: '#f0fdf4',   // Light emerald tint for AI
    100: '#d1fae5',
    200: '#a7f3d0',
    500: '#059669',  // Forest Green (Primary)
    600: '#047857',  // hover
    700: '#065f46',  // active
    900: '#064e3b',  // Deep Emerald (Surface)
  },
  gray: {
    50: '#fffcf5',   // Warm White/Sand (Page Background)
    100: '#fcfaf2',  // Card Background
    200: '#f5f5f4',  // Separator
    300: '#e7e5e4',  // Border (Stone)
    400: '#d6d3d1',
    500: '#78716c',  // Secondary Text (Stone Gray)
    700: '#44403c',  // Main Text (Stone Black)
    900: '#292524',  // Headings
  },
  // ... semantic colors remain similar but adjusted to stone palette
};

export const borderRadius = {
  none: '0',
  sm: '6px',    // Refined small radius
  md: '12px',   // Refined standard radius
  lg: '16px',
  xl: '24px',
  full: '9999px',
};

export const spacing = {
  xs: '4px',
  sm: '8px',
  md: '12px',   // Tightened md (was 16px)
  lg: '16px',   // Tightened lg (was 24px)
  xl: '24px',   // Tightened xl (was 32px)
  '2xl': '32px',
  '3xl': '48px',
};
```

- [ ] **Step 2: Sync Ant Design Theme**
Update `web/src/theme/antd-theme.ts` to use the new tokens.

```typescript
// web/src/theme/antd-theme.ts (partial update)
export const antdTheme: ThemeConfig = {
  token: {
    colorPrimary: '#059669',
    colorBgLayout: '#fffcf5',
    colorBgContainer: '#ffffff',
    colorText: '#292524',
    colorTextSecondary: '#78716c',
    colorBorder: '#e7e5e4',
    borderRadius: 12,
    fontSizeHeading1: 31, // Outfit scale
    fontFamily: "'DM Sans', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'sans-serif'",
    // ... update other mappings
  },
  components: {
    Layout: {
      siderBg: '#064e3b',
      headerBg: '#ffffff',
    },
    Menu: {
      itemSelectedBg: 'rgba(5, 150, 105, 0.15)',
      itemSelectedColor: '#059669',
      itemBorderRadius: 12, // Rectangular with 12px radius
    },
    Card: {
      borderRadiusLG: 12,
      paddingLG: 16, // High density
    }
  }
};
```

- [ ] **Step 3: Update Tailwind Configuration**
Sync `web/tailwind.config.js` with the new token values.

- [ ] **Step 4: Commit**
```bash
git add web/src/design-system/tokens.ts web/src/theme/antd-theme.ts web/tailwind.config.js
git commit -m "style: update global design tokens for Calm Emerald theme"
```

---

### Task 2: Refactor Layout Foundation

**Files:**
- Modify: `web/src/app/layout/AppLayout.tsx`

- [ ] **Step 1: Update Sidebar (Sider) Styling**
Apply the Deep Emerald background and refined selection states. Ensure vertical margins between menu items are tightened to `4px`.

- [ ] **Step 2: Update Header Styling**
Change header background and border. Ensure the search input uses the new stone border and 6px radius.

- [ ] **Step 3: Update Page Content Wrapper**
Apply the new warm white background and high-density padding (`p-4` instead of `p-6` on large screens).

- [ ] **Step 4: Commit**
```bash
git add web/src/app/layout/AppLayout.tsx
git commit -m "style: apply high-density emerald layout foundation"
```

---

### Task 3: Migrate Dashboard Page (High Density)

**Files:**
- Modify: `web/src/pages/Dashboard/Dashboard.tsx`

- [ ] **Step 1: Update Page Spacing**
Change `space-y-6` to `space-y-4` in the main container.

- [ ] **Step 2: Tighten Grid Gutters**
Update `<Row gutter={[16, 16]}>` to `<Row gutter={[12, 12]}>` for all rows in the dashboard.

- [ ] **Step 3: Commit**
```bash
git add web/src/pages/Dashboard/Dashboard.tsx
git commit -m "style: tighten dashboard page layout density"
```

---

### Task 4: Refactor Dashboard Components

**Files:**
- Modify: `web/src/components/Dashboard/HealthCard.tsx`
- Modify: `web/src/components/Dashboard/WorkloadHealthCard.tsx`
- Modify: `web/src/components/Dashboard/OperationsCard.tsx`
- Modify: `web/src/components/Dashboard/AlertPanel.tsx`

- [ ] **Step 1: Refactor HealthCard**
Update padding to 16px, border to stone, and shadow to Level 1. Ensure the progress bar uses Forest Green.

- [ ] **Step 2: Refactor WorkloadHealthCard**
Update padding and internal spacing between progress items.

- [ ] **Step 3: Refactor OperationsCard & AlertPanel**
Apply high-density padding and stone borders.

- [ ] **Step 4: Commit**
```bash
git add web/src/components/Dashboard/HealthCard.tsx web/src/components/Dashboard/WorkloadHealthCard.tsx web/src/components/Dashboard/OperationsCard.tsx web/src/components/Dashboard/AlertPanel.tsx
git commit -m "style: migrate dashboard cards to high-density emerald style"
```

---

### Task 5: AI Activity Card Refresh

**Files:**
- Modify: `web/src/components/Dashboard/AIActivityCard.tsx`

- [ ] **Step 1: Apply Forest Green Theme**
Remove any remaining blue/purple accents. Use Forest Green for AI icons and success metrics.

- [ ] **Step 2: Emerald Tint for AI Areas**
Apply `#f0fdf4` (Primary 50) as a background for AI stats or session highlights to subtly distinguish them.

- [ ] **Step 3: Commit**
```bash
git add web/src/components/Dashboard/AIActivityCard.tsx
git commit -m "style: update AI Activity card with organic emerald palette"
```

---

### Task 6: Final Verification

- [ ] **Step 1: Run Visual Regression (Manual)**
Start the dev server and verify the layout, colors, and density on both Desktop and Mobile views.
Command: `npm run dev` (in web directory)

- [ ] **Step 2: Verify existing tests pass**
Run: `npm test web/src/components/Dashboard/HealthCard.test.tsx`
Expected: PASS (if tests only check logic, not specific CSS colors that were changed)

- [ ] **Step 3: Final Commit & Cleanup**
```bash
git commit -m "style: complete Calm Emerald UI redesign"
```
