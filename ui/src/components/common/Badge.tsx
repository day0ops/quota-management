import styled from '@emotion/styled';
import { colors, radius, spacing, fontSize } from '../../styles';

type BadgeVariant = 'default' | 'success' | 'warning' | 'error' | 'info';

interface BadgeProps {
  variant?: BadgeVariant;
}

const variantStyles = {
  default: {
    background: colors.secondary,
    color: colors.foreground,
    border: colors.border,
  },
  success: {
    background: colors.successBg,
    color: colors.success,
    border: colors.successBorder,
  },
  warning: {
    background: colors.warningBg,
    color: colors.warning,
    border: colors.warningBorder,
  },
  error: {
    background: colors.errorBg,
    color: colors.error,
    border: colors.errorBorder,
  },
  info: {
    background: colors.infoBg,
    color: colors.info,
    border: colors.infoBorder,
  },
};

export const Badge = styled.span<BadgeProps>`
  display: inline-flex;
  align-items: center;
  padding: ${spacing[1]} ${spacing[2]};
  border-radius: ${radius.full};
  font-size: ${fontSize.xs};
  font-weight: 500;

  ${({ variant = 'default' }) => {
    const style = variantStyles[variant];
    return `
      background: ${style.background};
      color: ${style.color};
      border: 1px solid ${style.border};
    `;
  }}
`;
