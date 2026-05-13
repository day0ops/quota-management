import { useState, useRef, useEffect } from 'react';
import { createPortal } from 'react-dom';
import styled from '@emotion/styled';
import { colors, radius, spacing, fontSize } from '../../styles';

interface InputWrapperProps {
  fullWidth?: boolean;
}

export const InputWrapper = styled.div<InputWrapperProps>`
  display: flex;
  flex-direction: column;
  gap: ${spacing[1]};
  ${({ fullWidth }) => fullWidth && 'width: 100%;'}
`;

export const Label = styled.label`
  font-size: ${fontSize.sm};
  font-weight: 500;
  color: ${colors.foreground};
`;

export const Input = styled.input`
  height: 40px;
  padding: 0 ${spacing[3]};
  background: ${colors.cardBg};
  border: 1px solid ${colors.border};
  border-radius: ${radius.md};
  color: ${colors.foreground};
  font-size: ${fontSize.sm};
  transition: border-color 0.15s ease;

  &::placeholder {
    color: ${colors.dimForeground};
  }

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
`;

export const Textarea = styled.textarea`
  min-height: 100px;
  padding: ${spacing[3]};
  background: ${colors.cardBg};
  border: 1px solid ${colors.border};
  border-radius: ${radius.md};
  color: ${colors.foreground};
  font-size: ${fontSize.sm};
  resize: vertical;
  transition: border-color 0.15s ease;

  &::placeholder {
    color: ${colors.dimForeground};
  }

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
`;

export const InputError = styled.span`
  font-size: ${fontSize.xs};
  color: ${colors.error};
`;

const LabelRow = styled.div`
  display: flex;
  align-items: center;
  gap: ${spacing[1]};
`;

const TooltipWrapper = styled.div`
  display: inline-flex;
  align-items: center;
`;

const TooltipIcon = styled.span`
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
  cursor: help;
`;

const TooltipContent = styled.div<{ $x: number; $y: number; $visible: boolean }>`
  position: fixed;
  top: ${({ $y }) => $y}px;
  left: ${({ $x }) => $x}px;
  transform: translate(-50%, -100%);
  padding: ${spacing[2]} ${spacing[3]};
  background: var(--color-tooltip-bg);
  color: var(--color-tooltip-text);
  font-size: ${fontSize.xs};
  line-height: 1.4;
  border-radius: ${radius.md};
  white-space: pre-line;
  width: 250px;
  max-width: 300px;
  opacity: ${({ $visible }) => ($visible ? 1 : 0)};
  visibility: ${({ $visible }) => ($visible ? 'visible' : 'hidden')};
  transition:
    opacity 0.15s ease,
    visibility 0.15s ease;
  z-index: 10000;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  pointer-events: none;

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

interface TooltipProps {
  content: string;
}

function Tooltip({ content }: TooltipProps) {
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
      <TooltipIcon
        ref={iconRef}
        onMouseEnter={() => setVisible(true)}
        onMouseLeave={() => setVisible(false)}
      >
        ?
      </TooltipIcon>
      {createPortal(
        <TooltipContent $x={position.x} $y={position.y} $visible={visible}>
          {content}
        </TooltipContent>,
        document.body
      )}
    </TooltipWrapper>
  );
}

interface FormFieldProps {
  label: string;
  error?: string;
  tooltip?: string;
  children: React.ReactNode;
  fullWidth?: boolean;
}

export function FormField({ label, error, tooltip, children, fullWidth }: FormFieldProps) {
  return (
    <InputWrapper fullWidth={fullWidth}>
      <LabelRow>
        <Label>{label}</Label>
        {tooltip && <Tooltip content={tooltip} />}
      </LabelRow>
      {children}
      {error && <InputError>{error}</InputError>}
    </InputWrapper>
  );
}
