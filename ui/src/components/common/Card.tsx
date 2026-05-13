import styled from '@emotion/styled';
import { colors, spacing } from '../../styles';

type CardVariant = 'default' | 'violet' | 'green' | 'fuchsia';

interface CardProps {
  padding?: keyof typeof spacing;
  variant?: CardVariant;
}

const getCardStyles = (variant: CardVariant) => {
  const configs = {
    default: {
      cardBg: 'linear-gradient(var(--color-card-bg), var(--color-card-bg))',
      borderGradient:
        'linear-gradient(135deg, rgba(128, 35, 195, 0.22) 0%, rgba(43, 34, 57, 0.06) 100%)',
    },
    violet: {
      cardBg: 'radial-gradient(circle at 135% 150%, rgba(128, 35, 195, 0.22) 0%, #11101C 65%)',
      borderGradient:
        'linear-gradient(135deg, rgba(128, 35, 195, 0.55) 0%, rgba(43, 34, 57, 0.15) 100%)',
    },
    green: {
      cardBg: 'radial-gradient(circle at 135% 150%, rgba(67, 255, 170, 0.16) 0%, #11101C 65%)',
      borderGradient:
        'linear-gradient(135deg, rgba(34, 197, 94, 0.5) 0%, rgba(43, 34, 57, 0.12) 100%)',
    },
    fuchsia: {
      cardBg: 'radial-gradient(circle at 135% 150%, rgba(220, 67, 255, 0.16) 0%, #11101C 65%)',
      borderGradient:
        'linear-gradient(135deg, rgba(220, 67, 255, 0.5) 0%, rgba(43, 34, 57, 0.12) 100%)',
    },
  };

  const cfg = configs[variant];
  return `
    background: ${cfg.cardBg} padding-box, ${cfg.borderGradient} border-box;
    border: 1px solid transparent;
    box-shadow: 0 3.364px 3.364px 0 rgba(0, 0, 0, 0.25);
  `;
};

export const Card = styled.div<CardProps>`
  border-radius: 16px;
  padding: ${({ padding = 6 }) => spacing[padding as keyof typeof spacing]};
  ${({ variant = 'default' }) => getCardStyles(variant)}
`;

export const CardHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: ${spacing[4]};
`;

export const CardTitle = styled.h2`
  font-size: 18px;
  font-weight: 500;
  color: ${colors.foreground};
`;

export const CardDescription = styled.p`
  font-size: 14px;
  color: ${colors.mutedForeground};
  margin-top: ${spacing[1]};
`;

export const CardContent = styled.div``;

export const CardFooter = styled.div`
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: ${spacing[3]};
  margin-top: ${spacing[4]};
  padding-top: ${spacing[4]};
  border-top: 1px solid ${colors.border};
`;
