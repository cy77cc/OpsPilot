import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import {
  CardGridSkeleton,
  DetailSkeleton,
  FormSkeleton,
  PageSkeleton,
  TableSkeleton,
} from './index';
import { renderWithAntd } from '../../test/utils/render';

describe('LoadingSkeleton semantic kit', () => {
  it('renders the page skeleton from the barrel export', () => {
    renderWithAntd(<PageSkeleton />);

    expect(screen.getByTestId('page-skeleton')).toBeInTheDocument();
  });

  it('renders a table skeleton with toolbar and rows', () => {
    renderWithAntd(<TableSkeleton toolbar rows={6} columns={5} />);

    expect(screen.getByTestId('table-skeleton')).toBeInTheDocument();
    expect(screen.getAllByTestId('table-skeleton-row')).toHaveLength(6);
  });

  it('renders a detail skeleton with summary cards', () => {
    renderWithAntd(<DetailSkeleton summaryCards={3} sections={2} />);

    expect(screen.getByTestId('detail-skeleton')).toBeInTheDocument();
    expect(screen.getAllByTestId('detail-skeleton-card')).toHaveLength(3);
  });

  it('renders a form skeleton with grouped fields and actions', () => {
    renderWithAntd(<FormSkeleton groups={2} actions />);

    expect(screen.getByTestId('form-skeleton')).toBeInTheDocument();
    expect(screen.getAllByTestId('form-skeleton-group')).toHaveLength(2);
    expect(screen.getByTestId('form-skeleton-actions')).toBeInTheDocument();
  });

  it('renders a card grid skeleton with repeated cards', () => {
    renderWithAntd(<CardGridSkeleton cards={4} columns={2} />);

    expect(screen.getByTestId('card-grid-skeleton')).toBeInTheDocument();
    expect(screen.getAllByTestId('card-grid-skeleton-card')).toHaveLength(4);
  });
});
