# OpsPilot UI Redesign: Calm Emerald (High-Density)

## 1. Visual Philosophy
OpsPilot shifts from a generic industrial look to a "Nature-inspired, High-Density" professional aesthetic. The goal is to reduce eye strain during long monitoring sessions while maintaining the precision and efficiency required for DevOps operations.

## 2. Color Palette
- **Primary Background:** `#fffcf5` (Warm White/Sand) - Reduces glare and blue light fatigue.
- **Brand Primary (Action):** `#059669` (Forest Green) - Used for primary buttons, active states, and success indicators.
- **Surface/Header:** `#064e3b` (Deep Emerald) - Used for navigation backgrounds to provide high contrast and authority.
- **Text (Primary):** `#292524` (Stone Black) - High readability on sand background.
- **Text (Secondary):** `#78716c` (Stone Gray) - For labels and metadata.
- **Borders:** `#e7e5e4` (Light Stone) - Subtle separation for high-density components.
- **Semantic Success:** `#059669` (Forest Green).
- **Semantic Warning:** `#d97706` (Amber).
- **Semantic Error:** `#b91c1c` (Deep Red).

## 3. Typography
- **Headings:** `Outfit` (Weight 600) - Geometric and modern, used for page titles and section headers.
- **Body/UI:** `DM Sans` (Weight 400/500) - Geometric sans-serif used for all functional text and navigation.
- **Data/Technical:** `Roboto Mono` - Used for metrics, IP addresses, terminal output, and code blocks.

## 4. Geometry & Spacing (High-Density Focus)
- **Border Radius:** 
  - Standard Components (Cards, Modals): `12px`
  - Small Elements (Inputs, Buttons, Tags): `6px`
  - Full: `9999px` (Status badges only)
- **Spacing Scale:**
  - Tight: `4px`, `8px`
  - Standard: `12px`, `16px`
  - Section: `24px` (Maximum gap between major layout blocks)
- **Shadows:** 
  - Level 1: `0 1px 2px 0 rgba(120, 113, 108, 0.02)`
  - Level 2: `0 4px 12px -2px rgba(120, 113, 108, 0.08)` (Used for elevated cards)

## 5. Component Execution
- **Sidebar (Sider):** Deep Emerald background. Active menu items use a Forest Green background with `12px` border radius (not pill-shaped). Items are packed closely with `4px` vertical margin.
- **App Header:** Pure white or sand background with a single-pixel stone border at the bottom. Minimal height (56px-64px).
- **Cards:** White background with `12px` radius and `1px` stone border. Internal padding fixed at `16px`.
- **Tables:** 
  - Mode: Compact.
  - Header: Light sand background, no vertical borders.
  - Hover: Subtle stone-wash effect.
- **AI Copilot:** Branded with Forest Green. Content surfaces use a light emerald tint (`#f0fdf4`) to distinguish AI-generated suggestions without breaking the organic palette.

## 6. Implementation Strategy
1. **Global Tokens:** Update `tailwind.config.js` and `antd-theme.ts` with the new color palette and radii.
2. **Layout Foundation:** Refactor `AppLayout.tsx` to apply the Deep Emerald sidebar and tight spacing.
3. **Common Components:** Update Button, Card, and Input styles.
4. **Dashboard Migration:** Update the main Dashboard cards to use the 12px/16px padding high-density layout.
