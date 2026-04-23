# Refactor Page Titles Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove redundant `h1` and `p` tags in specified page components.

**Architecture:** Use `replace` tool to surgically remove the title blocks. If title is with buttons, change `justify-between` to `justify-end` and remove title column. If title is alone, remove the whole header div.

**Tech Stack:** React, TypeScript, Tailwind CSS.

---

### Task 1: Refactor Observability Pages

**Files:**
- web/src/pages/Deployment/Observability/AIOpsInsightsPage.tsx
- web/src/pages/Deployment/Observability/AuditLogsPage.tsx

- [ ] **Step 1: Refactor AIOpsInsightsPage.tsx**
- [ ] **Step 2: Refactor AuditLogsPage.tsx**

### Task 2: Refactor Deployment Creation Pages

**Files:**
- web/src/pages/Deployment/DeploymentCreatePage.tsx
- web/src/pages/Deployment/EnhancedDeploymentCreatePage.tsx

- [ ] **Step 1: Refactor DeploymentCreatePage.tsx**
- [ ] **Step 2: Refactor EnhancedDeploymentCreatePage.tsx**

### Task 3: Refactor Deployment Detail and List Pages

**Files:**
- web/src/pages/Deployment/DeploymentDetailPage.tsx
- web/src/pages/Deployment/DeploymentListPage.tsx
- web/src/pages/Deployment/DeploymentOverviewPage.tsx
- web/src/pages/Deployment/ApprovalCenterPage.tsx

- [ ] **Step 1: Refactor DeploymentDetailPage.tsx**
- [ ] **Step 2: Refactor DeploymentListPage.tsx**
- [ ] **Step 3: Refactor DeploymentOverviewPage.tsx**
- [ ] **Step 4: Refactor ApprovalCenterPage.tsx**

### Task 4: Refactor Deployment Targets Pages

**Files:**
- web/src/pages/Deployment/Targets/DeploymentTargetListPage.tsx
- web/src/pages/Deployment/Targets/EnvironmentBootstrapWizard.tsx
- web/src/pages/Deployment/Targets/CreateTargetWizard.tsx
- web/src/pages/Deployment/Targets/DeploymentTargetDetailPage.tsx

- [ ] **Step 1: Refactor DeploymentTargetListPage.tsx**
- [ ] **Step 2: Refactor EnvironmentBootstrapWizard.tsx**
- [ ] **Step 3: Refactor CreateTargetWizard.tsx**
- [ ] **Step 4: Refactor DeploymentTargetDetailPage.tsx**

### Task 5: Refactor Infrastructure Pages

**Files:**
- web/src/pages/Deployment/Infrastructure/ClusterBootstrapWizard.tsx
- web/src/pages/Deployment/Infrastructure/CredentialListPage.tsx
- web/src/pages/Deployment/Infrastructure/ClusterListPage.tsx
- web/src/pages/Deployment/Infrastructure/ClusterImportWizard.tsx

- [ ] **Step 1: Refactor ClusterBootstrapWizard.tsx**
- [ ] **Step 2: Refactor CredentialListPage.tsx**
- [ ] **Step 3: Refactor ClusterListPage.tsx**
- [ ] **Step 4: Refactor ClusterImportWizard.tsx**

### Task 6: Refactor Hosts and Services Pages

**Files:**
- web/src/pages/Hosts/HostListPage.tsx
- web/src/pages/Services/ServiceProvisionPage.tsx
- web/src/pages/Services/ServiceDetailPage.tsx
- web/src/pages/Services/ServiceListPage.tsx

- [ ] **Step 1: Refactor HostListPage.tsx**
- [ ] **Step 2: Refactor ServiceProvisionPage.tsx**
- [ ] **Step 3: Refactor ServiceDetailPage.tsx**
- [ ] **Step 4: Refactor ServiceListPage.tsx**
