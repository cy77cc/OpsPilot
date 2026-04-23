# Design: Refactor Page Titles to Global Header

The page titles in the OpsPilot web application are being moved to a global header. This refactoring removes the redundant local `h1` titles and descriptions from individual page components.

## Approaches Considered

### 1. Minimal Removal
- Only remove the `h1` and `p` tags.
- Leaves the container `div` and any action buttons in place.
- Pro: Simplest change.
- Con: May leave empty `div`s or awkward `justify-between` layouts with nothing on the left.

### 2. Layout Adjustment (Recommended)
- Remove `h1` and `p`.
- If they were in a `flex justify-between` container with actions on the right, change the container to `flex justify-end`.
- If the container only held the title, remove the container entirely while ensuring vertical spacing (margins) is maintained if needed.
- Pro: Cleaner UI, follows modern dashboard patterns.
- Con: Requires more careful adjustment of each file.

## Design Details

### Header Patterns

Most pages follow this pattern:
```tsx
<div className="flex items-center justify-between mb-6">
  <div>
    <h1 className="...">Title</h1>
    <p className="...">Description</p>
  </div>
  <Space>
    <Button>Action</Button>
  </Space>
</div>
```

Refactored pattern:
```tsx
<div className="flex justify-end mb-6">
  <Space>
    <Button>Action</Button>
  </Space>
</div>
```

If no actions exist:
```tsx
<div className="mb-6">
  {/* Entire header div removed if it only had title/description */}
</div>
```

## Affected Files
1. web/src/pages/Deployment/Observability/PolicyManagementPage.tsx
2. web/src/pages/Deployment/Observability/DeploymentTopologyPage.tsx
3. web/src/pages/Deployment/DeploymentCreatePage.tsx
4. web/src/pages/Deployment/Observability/MetricsDashboardPage.tsx
5. web/src/pages/Deployment/Observability/AIOpsInsightsPage.tsx
6. web/src/pages/Deployment/Observability/AuditLogsPage.tsx
7. web/src/pages/Deployment/EnhancedDeploymentCreatePage.tsx
8. web/src/pages/Deployment/DeploymentDetailPage.tsx
9. web/src/pages/Deployment/Targets/DeploymentTargetListPage.tsx
10. web/src/pages/Deployment/Targets/EnvironmentBootstrapWizard.tsx
11. web/src/pages/Deployment/Targets/CreateTargetWizard.tsx
12. web/src/pages/Deployment/Targets/DeploymentTargetDetailPage.tsx
13. web/src/pages/Deployment/DeploymentListPage.tsx
14. web/src/pages/Deployment/Infrastructure/ClusterBootstrapWizard.tsx
15. web/src/pages/Deployment/Infrastructure/CredentialListPage.tsx
16. web/src/pages/Deployment/Infrastructure/ClusterListPage.tsx
17. web/src/pages/Deployment/Infrastructure/ClusterImportWizard.tsx
18. web/src/pages/Deployment/ApprovalCenterPage.tsx
19. web/src/pages/Deployment/DeploymentOverviewPage.tsx
20. web/src/pages/Hosts/HostListPage.tsx
21. web/src/pages/Services/ServiceProvisionPage.tsx
22. web/src/pages/Services/ServiceDetailPage.tsx
23. web/src/pages/Services/ServiceListPage.tsx
