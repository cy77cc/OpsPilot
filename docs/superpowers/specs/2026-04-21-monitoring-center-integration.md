# Monitoring Center Integration Spec

## 1. Introduction
This spec outlines the redesign of the entire Monitoring & Alerting module into a unified "Monitoring Center". The goal is to provide a cohesive, high-density experience that integrates real-time dashboarding with configuration management.

## 2. Architectural Design

### 2.1. Unified Layout (MonitorCenterLayout)
- **Side Navigation**: A compact, dark-themed sidebar (180px - 200px) providing quick access to:
    - Dashboard (Monitor)
    - Alert History
    - Rules Config
    - Notification Channels
    - Routing Policies
    - Delivery Records
- **Global Header Stats**: A thin row at the top of the content area showing key live metrics (Active Alerts, Pending Records, Total Rules, System Health).
- **Nested Routing**: Use `<Outlet />` to render sub-pages within the layout.

### 2.2. Page Redesign Principles
- **Information Density**: 
    - Use Ant Design `size="small"` for all tables and cards.
    - Reduce whitespace and margins.
    - Use compact indicator dots (Badges) for status instead of large tags.
- **Contextual Actions**:
    - "Sync Rules" button resides exclusively in the Rules Config page header.
    - "Reload" button in the Dashboard for metrics refresh.

## 3. Component Details

### 3.1. Layout Component (`MonitorCenterLayout.tsx`)
- Fetches global stats (e.g., active alert count) to display in the header.
- Manages the sidebar state and navigation.

### 3.2. Dashboard Page (`MonitorPage.tsx`)
- Removed configuration navigation buttons.
- Features a more compact layout for resource usage bars and the alert trend chart.
- Includes a "Current Active Alerts" table directly on the dashboard for immediate visibility.

### 3.3. Config Pages (Rules, Channels, Routes, Deliveries)
- Updated to fit within the new layout.
- Standardized header format: `[Title] [Scope/Filters] [Primary Action]`.
- All tables set to `size="small"`.

## 4. Implementation Strategy

### 4.1. Routing Changes
- Group all `/monitor/*` routes under the `MonitorCenterLayout`.

### 4.2. File Modifications
- **Create**: `web/src/pages/Monitor/MonitorCenterLayout.tsx`
- **Modify**: `web/src/app/routes/observability.routes.tsx`
- **Modify**: `web/src/pages/Monitor/MonitorPage.tsx`
- **Modify**: `web/src/pages/Monitor/AlertsPage.tsx`
- **Modify**: `web/src/pages/Monitor/RulesConfigPage.tsx`
- **Modify**: `web/src/pages/Monitor/ChannelsConfigPage.tsx`
- **Modify**: `web/src/pages/Monitor/RoutingConfigPage.tsx`
- **Modify**: `web/src/pages/Monitor/DeliveriesPage.tsx`

## 5. Verification Plan
- Verify that navigating between monitoring sub-pages is seamless and doesn't re-render the sidebar/header unnecessarily.
- Confirm "Sync Rules" works correctly in its new location.
- Check that global stats in the header update correctly (or on refresh).
- Ensure mobile responsiveness (sidebar should collapse or hide).
