import React, { useState } from 'react';

interface CollapsibleProps {
  children: React.ReactNode;
  trigger: React.ReactNode;
  defaultOpen?: boolean;
  className?: string;
}

/**
 * 可折叠组件 (静态)
 */
const Collapsible: React.FC<CollapsibleProps> = ({
  children,
  trigger,
  defaultOpen = false,
  className = '',
}) => {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  return (
    <div className={className}>
      <div
        onClick={() => setIsOpen(!isOpen)}
        className="cursor-pointer"
      >
        {trigger}
      </div>

      {isOpen && (
        <div style={{ overflow: 'hidden' }}>
          {children}
        </div>
      )}
    </div>
  );
};

export default Collapsible;
