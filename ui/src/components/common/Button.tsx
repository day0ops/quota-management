import styled from '@emotion/styled';
import { colors, radius, spacing, fontSize, fontWeight } from '../../styles';

type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'ghost';
type ButtonSize = 'sm' | 'md' | 'lg';

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  fullWidth?: boolean;
}

const variantStyles = {
  primary: {
    background: 'linear-gradient(117deg, #6844ff -23.54%, #1d283a 223.49%)',
    color: '#FAFAFA',
    border: 'none',
    hover: 'linear-gradient(117deg, #382b93 -23.54%, #2b2568 223.49%)',
    active: 'linear-gradient(117deg, #31267a -23.54%, #282658 223.49%)',
  },
  secondary: {
    background: colors.secondary,
    color: colors.foreground,
    border: `1px solid ${colors.border}`,
    hover: colors.secondaryHover,
    active: colors.secondaryActive,
  },
  danger: {
    background: colors.error,
    color: '#FFFFFF',
    border: 'none',
    hover: '#DC2626',
    active: '#B91C1C',
  },
  ghost: {
    background: 'transparent',
    color: colors.mutedForeground,
    border: 'none',
    hover: colors.hoverBg,
    active: colors.activeBg,
  },
};

const sizeStyles = {
  sm: {
    padding: `${spacing[1]} ${spacing[2]}`,
    fontSize: fontSize.sm,
    height: '32px',
  },
  md: {
    padding: `${spacing[2]} ${spacing[4]}`,
    fontSize: fontSize.sm,
    height: '40px',
  },
  lg: {
    padding: `${spacing[3]} ${spacing[5]}`,
    fontSize: fontSize.md,
    height: '48px',
  },
};

export const Button = styled.button<ButtonProps>`
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: ${spacing[2]};
  border-radius: ${radius.full};
  font-weight: ${fontWeight.medium};
  transition: all 0.15s ease;
  white-space: nowrap;
  cursor: pointer;

  ${({ variant = 'primary' }) => {
    const style = variantStyles[variant];
    return `
      background: ${style.background};
      color: ${style.color};
      border: ${style.border};

      &:hover:not(:disabled) {
        background: ${style.hover};
      }

      &:active:not(:disabled) {
        background: ${style.active};
      }
    `;
  }}

  ${({ size = 'md' }) => {
    const style = sizeStyles[size];
    return `
      padding: ${style.padding};
      font-size: ${style.fontSize};
      height: ${style.height};
    `;
  }}

  ${({ fullWidth }) => fullWidth && 'width: 100%;'}

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  &:focus-visible {
    outline: 2px solid ${colors.primary};
    outline-offset: 2px;
  }
`;
