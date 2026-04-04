# Loading Skeleton Unification Design

## Goal

Unify frontend loading behavior so route changes do not blank the entire app shell, and data-loading states consistently use skeletons instead of mixed full-page fallbacks, spinners, and Ant Design default loading placeholders.

## Problems

The current frontend has three user-facing loading problems:

1. Route-level lazy loading is wrapped by a top-level `Suspense` in `ProtectedApp.tsx`, so navigating between some menu items can replace the whole app with a loading screen instead of keeping the sidebar and header stable.
2. Page and component loading patterns are inconsistent. The codebase mixes `Spin`, `Card loading`, `Table loading`, ad hoc `if (loading) return ...` branches, and a small amount of custom skeleton usage.
3. Even when loading is technically local, the placeholder shape often does not match the eventual layout, producing visual jumps and uneven perceived performance.

## Design Goals

- Keep the app shell stable during route transitions.
- Use skeletons as the default content-loading pattern across the app.
- Preserve `Button loading` for action feedback such as submit, refresh, and retry.
- Replace large blocking spinners with layout-matched skeletons wherever content is loading.
- Make the loading system composable so new pages can adopt a standard pattern instead of inventing their own.

## Non-Goals

- No redesign of page information architecture or visual theme.
- No attempt to remove all loading indicators in a single pass for tiny inline async states that are not content placeholders.
- No backend or API behavior changes.

## Loading Model

The app will use two coordinated loading layers.

### 1. Route Content Loading

The app shell remains mounted at all times. Route lazy loading only affects the main content region.

- `AppLayout` stays visible during navigation.
- The content slot inside the layout shows a page-level skeleton while the route chunk resolves.
- Route transitions should never replace the entire viewport with a generic "加载中..." screen.

### 2. Page and Component Data Loading

Once a route has mounted, data fetching uses semantic skeletons that match the page structure.

- Dashboard-style pages show stat-card and chart skeletons.
- List pages show header, filters, and table skeletons.
- Detail pages show title, metadata, and content-section skeletons.
- Form pages show grouped form skeletons.
- Drawers, tabs, and side panels use section/detail skeletons instead of blocking spinners.

## Component Architecture

The existing `components/LoadingSkeleton` module will be expanded into a small loading system.

### Base Layer

The current `LoadingSkeleton` component becomes the low-level primitive provider for repeated skeleton blocks and layout utilities.

### Semantic Components

Add semantic wrappers that express intent rather than raw shape:

- `PageSkeleton`
- `SectionSkeleton`
- `TableSkeleton`
- `DetailSkeleton`
- `FormSkeleton`
- `CardGridSkeleton`

These components should be lightweight wrappers around Ant Design `Skeleton` primitives plus shared spacing tokens. Their API should favor a few stable variants over highly dynamic configuration.

## Route Boundary Changes

`ProtectedApp.tsx` currently places a single `Suspense` around the lazy-loaded layout and the routed content. That causes full-page fallback behavior.

The route boundary should be split:

- Keep app-level providers unchanged.
- Keep `AppLayout` mounted once loaded.
- Move route-specific `Suspense` so fallback is rendered inside the content area rather than outside the shell.

Recommended structure:

- `ProtectedApp` resolves `AppLayout`
- `AppLayout` owns the content slot
- routed page content is wrapped with a route-level fallback such as `PageSkeleton`

If needed, a small helper can wrap route elements so the same fallback pattern is reused across routes.

## Replacement Rules

The following patterns should be standardized.

### Replace

- top-level route fallback text screens
- `if (loading) return <Spin ... />`
- `Card loading={loading}` for primary content sections
- `Table loading={loading}` for first-load and full-content reload states
- permission-loading large spinners

### Keep

- `Button loading` for action feedback
- very small inline pending indicators only where a skeleton would be disproportionate

## Page-Type Conventions

### Dashboard Pages

Use card-grid and chart skeletons that preserve card height and graph regions.

### List Pages

Use:

- header skeleton
- toolbar/filter skeleton
- table skeleton

Do not rely on table overlay spinners for page-first render.

### Detail Pages

Use:

- title and metadata skeleton
- summary-card skeleton
- multi-section content skeleton

### Form Pages

Use:

- page header skeleton
- grouped field skeleton
- preview or summary skeleton for multi-step flows when that region exists

Form submission keeps button-level loading.

## Permission Loading

`Authorized.tsx` currently uses a large `Spin` while permission state resolves. This should be replaced by a local content skeleton so permission checks feel consistent with route and data loading.

The fallback should match the page region, not the whole viewport.

## Migration Strategy

Implement in four passes.

### Pass 1. Route Boundary

- change the lazy-loading boundary so the app shell does not disappear
- introduce a standard page-level fallback

### Pass 2. Shared Skeleton Kit

- extend `components/LoadingSkeleton`
- add semantic skeleton variants
- define shared sizing and spacing rules

### Pass 3. Bulk Replacement

Replace the most visible loading patterns across major pages:

- dashboard
- deployment
- services
- hosts
- monitor
- settings
- cluster infrastructure

### Pass 4. Cleanup

- remove obsolete spinner-based patterns
- normalize remaining outliers in drawers, tabs, and detail sections

## Verification

At minimum, verify:

- TypeScript compiles
- route transitions keep header and sidebar visible
- list page first-load shows skeleton instead of blank or overlay spinner
- detail page first-load shows local skeleton instead of full-page spinner
- dashboard cards and charts show skeletons aligned with final layout
- submit and refresh buttons still use `Button loading`
- permission checks do not flash a full-page spinner

## Risks

### Layout Mismatch

If skeletons are too generic, route changes can feel more jumpy than before. Skeleton shapes must roughly match final structure.

### Double Loading States

Pages may temporarily render both custom skeletons and Ant Design built-in loading states. Part of the implementation is removing overlapping `loading` props where the skeleton already owns the state.

### Scope Creep

The codebase contains many loading call sites. The implementation should prioritize consistency rules and common primitives instead of building a bespoke skeleton for every screen.

## Recommendation

Proceed with the unified loading-kit approach rather than a batch spinner-to-skeleton replacement. The boundary fix and shared primitives solve both primary UX issues:

- the app shell no longer disappears on navigation
- content-loading visuals become consistent across the app
