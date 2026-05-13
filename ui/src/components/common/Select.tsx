import styled from '@emotion/styled';
import { colors, radius, spacing, fontSize } from '../../styles';

export const Select = styled.select`
  height: 40px;
  padding: 0 ${spacing[3]};
  padding-right: ${spacing[8]};
  background: ${colors.cardBg};
  border: 1px solid ${colors.border};
  border-radius: ${radius.md};
  color: ${colors.foreground};
  font-size: ${fontSize.sm};
  cursor: pointer;
  transition: border-color 0.15s ease;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%23A1A1AA' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;

  &:hover:not(:disabled) {
    border-color: ${colors.borderLight};
  }

  &:focus {
    outline: none;
    border-color: ${colors.primary};
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  option {
    background: ${colors.cardBg};
    color: ${colors.foreground};
  }
`;
