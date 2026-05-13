import styled from '@emotion/styled';
import { keyframes } from '@emotion/react';
import { colors } from '../../styles';

const spin = keyframes`
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
`;

interface SpinnerProps {
  size?: number;
}

export const Spinner = styled.div<SpinnerProps>`
  width: ${({ size = 20 }) => size}px;
  height: ${({ size = 20 }) => size}px;
  border: 2px solid ${colors.border};
  border-top-color: ${colors.primary};
  border-radius: 50%;
  animation: ${spin} 0.8s linear infinite;
`;

const LoadingContainer = styled.div`
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px;
`;

export function Loading() {
  return (
    <LoadingContainer>
      <Spinner size={32} />
    </LoadingContainer>
  );
}
