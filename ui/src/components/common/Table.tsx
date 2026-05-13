import styled from '@emotion/styled';
import { colors, spacing, fontSize } from '../../styles';

export const TableContainer = styled.div`
  overflow-x: auto;
  border: 1px solid ${colors.border};
  border-radius: 8px;
`;

export const Table = styled.table`
  width: 100%;
  border-collapse: collapse;
`;

export const TableHead = styled.thead`
  background: ${colors.surfaceBg};
`;

export const TableBody = styled.tbody``;

export const TableRow = styled.tr<{ clickable?: boolean }>`
  border-bottom: 1px solid ${colors.borderDark};

  &:last-child {
    border-bottom: none;
  }

  ${({ clickable }) =>
    clickable &&
    `
    cursor: pointer;
    &:hover {
      background: ${colors.tableRowHover};
    }
  `}
`;

export const TableHeader = styled.th<{ align?: 'left' | 'center' | 'right' }>`
  padding: ${spacing[3]} ${spacing[4]};
  text-align: ${({ align = 'left' }) => align};
  font-size: ${fontSize.xs};
  font-weight: 500;
  color: ${colors.mutedForeground};
  text-transform: uppercase;
  letter-spacing: 0.05em;
  white-space: nowrap;
  background: inherit;
`;

export const TableCell = styled.td<{ align?: 'left' | 'center' | 'right' }>`
  padding: ${spacing[3]} ${spacing[4]};
  text-align: ${({ align = 'left' }) => align};
  font-size: ${fontSize.sm};
  color: ${colors.foreground};
  vertical-align: middle;
`;

export const EmptyState = styled.div`
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: ${spacing[12]} ${spacing[6]};
  text-align: center;
  color: ${colors.mutedForeground};
`;

export const EmptyStateText = styled.p`
  font-size: ${fontSize.sm};
  margin-bottom: ${spacing[4]};
`;
