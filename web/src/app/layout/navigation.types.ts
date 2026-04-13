import type { ReactNode } from 'react';

export type MenuLeaf = {
  key: string;
  icon?: ReactNode;
  label: string;
};

export type MenuSection = {
  key: string;
  title: string;
  items: MenuLeaf[];
};

export type MenuPathEntry = {
  title: string;
  key?: string;
};
