import { useState, useRef, useEffect } from 'react';
import { createPortal } from 'react-dom';
import styled from '@emotion/styled';
import { colors, fontSize, spacing, radius } from '../../styles';

const TooltipWrapper = styled.span`
  display: inline-flex;
  align-items: center;
  gap: ${spacing[1]};
  cursor: help;
`;

const TooltipText = styled.div<{ $x: number; $y: number; $visible: boolean }>`
  position: fixed;
  top: ${({ $y }) => $y}px;
  left: ${({ $x }) => $x}px;
  transform: translate(-50%, -100%);
  padding: ${spacing[2]} ${spacing[3]};
  background: var(--color-tooltip-bg);
  color: var(--color-tooltip-text);
  font-size: ${fontSize.xs};
  font-weight: 400;
  text-transform: none;
  letter-spacing: normal;
  line-height: 1.4;
  max-width: 280px;
  white-space: pre-line;
  text-align: left;
  border-radius: ${radius.md};
  opacity: ${({ $visible }) => ($visible ? 1 : 0)};
  visibility: ${({ $visible }) => ($visible ? 'visible' : 'hidden')};
  transition:
    opacity 0.15s ease,
    visibility 0.15s ease;
  z-index: 10000;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
  pointer-events: none;

  /* Arrow */
  &::after {
    content: '';
    position: absolute;
    top: 100%;
    left: 50%;
    transform: translateX(-50%);
    border: 6px solid transparent;
    border-top-color: var(--color-tooltip-bg);
  }
`;

const HelpIcon = styled.span`
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: ${colors.border};
  color: ${colors.mutedForeground};
  font-size: 11px;
  font-weight: 600;
  flex-shrink: 0;
`;

interface TooltipProps {
  text: string;
  children: React.ReactNode;
}

export function Tooltip({ text, children }: TooltipProps) {
  const [visible, setVisible] = useState(false);
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const iconRef = useRef<HTMLSpanElement>(null);

  useEffect(() => {
    if (visible && iconRef.current) {
      const rect = iconRef.current.getBoundingClientRect();
      setPosition({
        x: rect.left + rect.width / 2,
        y: rect.top - 8,
      });
    }
  }, [visible]);

  return (
    <TooltipWrapper>
      {children}
      <HelpIcon
        ref={iconRef}
        onMouseEnter={() => setVisible(true)}
        onMouseLeave={() => setVisible(false)}
      >
        ?
      </HelpIcon>
      {createPortal(
        <TooltipText $x={position.x} $y={position.y} $visible={visible}>
          {text}
        </TooltipText>,
        document.body
      )}
    </TooltipWrapper>
  );
}
