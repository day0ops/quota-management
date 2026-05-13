import styled from '@emotion/styled';
import { colors, spacing, fontSize, radius } from '../../styles';

interface PaginationProps {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}

const PaginationContainer = styled.div`
  display: flex;
  align-items: center;
  gap: ${spacing[2]};
  flex-wrap: wrap;
`;

const PageInfo = styled.span`
  font-size: ${fontSize.sm};
  color: ${colors.mutedForeground};
  margin-right: ${spacing[2]};
`;

const PageButton = styled.button<{ active?: boolean; disabled?: boolean }>`
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  height: 32px;
  padding: 0 ${spacing[2]};
  border-radius: ${radius.md};
  font-size: ${fontSize.sm};
  border: 1px solid ${colors.border};
  cursor: ${({ disabled }) => (disabled ? 'not-allowed' : 'pointer')};
  transition: all 0.15s ease;
  background: ${({ active }) => (active ? colors.primary : colors.cardBg)};
  color: ${({ active }) => (active ? '#FFFFFF' : colors.mutedForeground)};
  opacity: ${({ disabled }) => (disabled ? 0.4 : 1)};

  &:hover:not(:disabled) {
    background: ${({ active }) => (active ? colors.primary : colors.hoverBg)};
    color: ${colors.foreground};
  }
`;

const Ellipsis = styled.span`
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  height: 32px;
  font-size: ${fontSize.sm};
  color: ${colors.mutedForeground};
`;

function getPageNumbers(page: number, totalPages: number): (number | 'ellipsis')[] {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, i) => i + 1);
  }

  const pages: (number | 'ellipsis')[] = [];

  if (page <= 4) {
    pages.push(1, 2, 3, 4, 5, 'ellipsis', totalPages);
  } else if (page >= totalPages - 3) {
    pages.push(
      1,
      'ellipsis',
      totalPages - 4,
      totalPages - 3,
      totalPages - 2,
      totalPages - 1,
      totalPages
    );
  } else {
    pages.push(1, 'ellipsis', page - 1, page, page + 1, 'ellipsis', totalPages);
  }

  return pages;
}

export function Pagination({ page, totalPages, onPageChange }: PaginationProps) {
  if (totalPages <= 1) return null;

  const pageNumbers = getPageNumbers(page, totalPages);

  return (
    <PaginationContainer>
      <PageInfo>
        Page {page} of {totalPages}
      </PageInfo>
      <PageButton disabled={page === 1} onClick={() => onPageChange(page - 1)}>
        &#8249;
      </PageButton>
      {pageNumbers.map((p, idx) =>
        p === 'ellipsis' ? (
          <Ellipsis key={`ellipsis-${idx}`}>&#8230;</Ellipsis>
        ) : (
          <PageButton key={p} active={p === page} onClick={() => onPageChange(p)}>
            {p}
          </PageButton>
        )
      )}
      <PageButton disabled={page === totalPages} onClick={() => onPageChange(page + 1)}>
        &#8250;
      </PageButton>
    </PaginationContainer>
  );
}
