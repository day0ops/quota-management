import styled from '@emotion/styled';
import { colors, radius } from '../../styles';

const Container = styled.div`
  width: 100%;
`;

const BarContainer = styled.div`
  height: 6px;
  background: ${colors.border};
  border-radius: ${radius.full};
  overflow: hidden;
`;

interface BarFillProps {
  percentage: number;
  variant: 'normal' | 'warning' | 'error';
}

const BarFill = styled.div<BarFillProps>`
  height: 100%;
  border-radius: ${radius.full};
  transition: width 0.3s ease;
  width: ${({ percentage }) => Math.min(percentage, 100)}%;
  background: ${({ variant }) => {
    switch (variant) {
      case 'error':
        return colors.error;
      case 'warning':
        return colors.warning;
      default:
        return 'linear-gradient(90deg, #6844FF 0%, #8023C3 100%)';
    }
  }};
`;

interface ProgressBarProps {
  value: number;
  max: number;
  warningThreshold?: number;
}

export function ProgressBar({ value, max, warningThreshold = 80 }: ProgressBarProps) {
  const percentage = max > 0 ? (value / max) * 100 : 0;
  const roundedPct = Math.round(percentage * 10) / 10;

  let variant: 'normal' | 'warning' | 'error' = 'normal';
  if (roundedPct >= 100) {
    variant = 'error';
  } else if (roundedPct >= warningThreshold) {
    variant = 'warning';
  }

  return (
    <Container>
      <BarContainer>
        <BarFill percentage={percentage} variant={variant} />
      </BarContainer>
    </Container>
  );
}
