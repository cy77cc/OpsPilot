# Monitoring Configuration Redesign Spec

## 1. Introduction
The goal of this redesign is to consolidate monitoring-related configuration pages (Rules, Channels, Routes, and Deliveries) into a single, cohesive "Monitoring Configuration Center". This will improve user efficiency by reducing page jumps, increasing information density, and placing actions (like "Sync Rules") in their logical context.

## 2. Proposed Changes

### 2.1. Layout: Monitoring Configuration Center
- **Component**: Create or update `MonitorConfigLayout` (or use a shared layout in `MonitorPage` or a new parent component).
- **Navigation**: Use a compact sidebar or top-level tabs to switch between:
    - Alert Rules (`/monitor/rules`)
    - Notification Channels (`/monitor/channels`)
    - Routing Policies (`/monitor/routing`)
    - Delivery Records (`/monitor/deliveries`)
- **Compatibility**: The existing routes in `observability.routes.tsx` will remain, but their components will be wrapped in this new layout or the layout will be integrated into each page consistently.

### 2.2. Information Density Improvements
- **Tables**: All `Table` components in these pages will be updated to use `size="small"`.
- **Spacing**: Use `Space` with `size="small"` for action buttons.
- **Headers**: Standardize the header section of each page to include a title, scope selector (where applicable), and primary actions in a single line.

### 2.3. Specific Page Adjustments

#### 2.3.1. Rules Config Page (`RulesConfigPage.tsx`)
- **Sync Rules Button**: Move the "Sync Rules" button from the main `MonitorPage` to the top-right toolbar of the `RulesConfigPage`.
- **Table Columns**: Optimize column widths to show more information without horizontal scrolling.

#### 2.3.2. Channels Config Page (`ChannelsConfigPage.tsx`)
- **Compact View**: Apply `size="small"` to the channels table.
- **Form Layout**: Ensure the creation/edit modal uses a compact layout.

#### 2.3.3. Routing Config Page (`RoutingConfigPage.tsx`)
- **Compact View**: Apply `size="small"` to the routing table.
- **Channel Display**: Improve how channel IDs are displayed (e.g., small tags instead of a raw JSON string if possible).

#### 2.3.4. Deliveries Page (`DeliveriesPage.tsx`)
- **Compact View**: Apply `size="small"` to the delivery records table.

## 3. Architecture & Components

### 3.1. File Structure
- `web/src/pages/Monitor/MonitorConfigLayout.tsx` (New): A wrapper component providing the sidebar/tabs navigation.
- `web/src/pages/Monitor/RulesConfigPage.tsx` (Modified): Update UI and add Sync logic.
- `web/src/pages/Monitor/ChannelsConfigPage.tsx` (Modified): Update UI.
- `web/src/pages/Monitor/RoutingConfigPage.tsx` (Modified): Update UI.
- `web/src/pages/Monitor/DeliveriesPage.tsx` (Modified): Update UI.
- `web/src/pages/Monitor/MonitorPage.tsx` (Modified): Remove redundant config buttons and focus on the dashboard metrics.

### 3.2. Navigation Implementation
We will use a shared `Layout` component that wraps the content of these pages. Alternatively, if they are separate routes, we can create a `MonitoringConfig` parent component with nested routes.

## 4. UI/UX Design

### 4.1. Visual Density
- Font size: Standard (14px) or slightly smaller (13px) for table content.
- Padding: Reduced padding in Cards and Tables.
- Consistency: All configuration pages will follow the same header pattern: `[Title] [Scope Selector] [Primary Action]`.

### 4.2. Action Flow
- When a user modifies a rule, they can immediately click "Sync Rules" located in the same view to apply changes to the monitoring system (e.g., Prometheus).

## 5. Implementation Plan
1. Create/refine the shared layout for monitoring config.
2. Update `RulesConfigPage.tsx`:
    - Apply `size="small"` to Table.
    - Add "Sync Rules" button and handler.
3. Update `ChannelsConfigPage.tsx`:
    - Apply `size="small"` to Table.
4. Update `RoutingConfigPage.tsx`:
    - Apply `size="small"` to Table.
5. Update `DeliveriesPage.tsx`:
    - Apply `size="small"` to Table.
6. Update `MonitorPage.tsx`:
    - Remove the configuration buttons from the header.
    - Ensure it remains a pure dashboard for metrics and active alerts.

## 6. Verification Plan
- **UI Check**: Verify that all tables are compact and visually consistent.
- **Functionality Check**: 
    - Verify "Sync Rules" works from the new location.
    - Verify all CRUD operations for rules, channels, and routes still work.
    - Verify navigation between config pages is seamless.
- **Regression**: Ensure the main `MonitorPage` dashboard still displays data correctly.
