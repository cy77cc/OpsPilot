/**
 * Design System Verification Test Component
 *
 * This component verifies that:
 * 1. Design tokens can be imported and used
 * 2. Tailwind classes work correctly
 * 3. Ant Design components use the custom theme
 */

import { Button, Card, Input, Tag } from 'antd';
import { colors, spacing } from '../design-system/tokens';

export const DesignSystemVerification = () => {
  return (
    <div className="p-8 space-y-8 bg-gray-50">
      {/* Section 1: Design Tokens Verification */}
      <Card title="1. Design Tokens Verification" className="shadow-md">
        <div className="space-y-4">
          <div>
            <h3 className="text-lg font-medium mb-2">Colors</h3>
            <div className="flex gap-4">
              <div
                className="w-20 h-20 rounded-lg"
                style={{ backgroundColor: colors.primary[500] }}
              />
              <div
                className="w-20 h-20 rounded-lg"
                style={{ backgroundColor: colors.success }}
              />
              <div
                className="w-20 h-20 rounded-lg"
                style={{ backgroundColor: colors.warning }}
              />
              <div
                className="w-20 h-20 rounded-lg"
                style={{ backgroundColor: colors.error }}
              />
            </div>
          </div>

          <div>
            <h3 className="text-lg font-medium mb-2">Spacing</h3>
            <div className="flex gap-2 items-end">
              <div
                className="bg-primary-500"
                style={{ width: spacing.xs, height: spacing.xs }}
              />
              <div
                className="bg-primary-500"
                style={{ width: spacing.sm, height: spacing.sm }}
              />
              <div
                className="bg-primary-500"
                style={{ width: spacing.md, height: spacing.md }}
              />
              <div
                className="bg-primary-500"
                style={{ width: spacing.lg, height: spacing.lg }}
              />
            </div>
          </div>
        </div>
      </Card>

      {/* Section 2: Tailwind Classes Verification */}
      <Card title="2. Tailwind Classes Verification" className="shadow-md">
        <div className="space-y-4">
          <div className="bg-primary-50 text-primary-700 p-4 rounded-lg">
            Primary 50 background with Primary 700 text
          </div>
          <div className="bg-gray-100 text-gray-700 p-md rounded-md shadow-sm">
            Gray 100 background with custom spacing and shadow
          </div>
          <div className="flex gap-sm">
            <div className="w-16 h-16 bg-success rounded-sm" />
            <div className="w-16 h-16 bg-warning rounded-md" />
            <div className="w-16 h-16 bg-error rounded-lg" />
          </div>
        </div>
      </Card>

      {/* Section 3: Ant Design Theme Verification */}
      <Card title="3. Ant Design Components Theme Verification" className="shadow-md">
        <div className="space-y-4">
          <div>
            <h3 className="text-lg font-medium mb-2">Buttons</h3>
            <div className="flex gap-4">
              <Button type="primary">Primary Button</Button>
              <Button>Default Button</Button>
              <Button type="dashed">Dashed Button</Button>
              <Button type="link">Link Button</Button>
            </div>
          </div>

          <div>
            <h3 className="text-lg font-medium mb-2">Input</h3>
            <Input placeholder="Test input with custom theme" />
          </div>

          <div>
            <h3 className="text-lg font-medium mb-2">Tags</h3>
            <div className="flex gap-2">
              <Tag color="success">Success</Tag>
              <Tag color="warning">Warning</Tag>
              <Tag color="error">Error</Tag>
              <Tag color="processing">Processing</Tag>
            </div>
          </div>

          <div>
            <h3 className="text-lg font-medium mb-2">Nested Card</h3>
            <Card size="small" className="bg-gray-50">
              This card should have rounded corners (12px) and proper shadow
            </Card>
          </div>
        </div>
      </Card>

      {/* Section 4: Animation Verification */}
      <Card title="4. Animation Classes Verification" className="shadow-md">
        <div className="space-y-4">
          <div className="animate-fade-in bg-primary-100 p-4 rounded-lg">
            Fade In Animation
          </div>
          <div className="animate-slide-in-up bg-success/10 p-4 rounded-lg">
            Slide In Up Animation
          </div>
          <div className="animate-scale-in bg-warning/10 p-4 rounded-lg">
            Scale In Animation
          </div>
        </div>
      </Card>
    </div>
  );
};
