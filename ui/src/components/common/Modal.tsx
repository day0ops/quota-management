import { useEffect } from 'react';
import styled from '@emotion/styled';
import { colors, radius, spacing, shadow } from '../../styles';

const Overlay = styled.div`
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: ${spacing[4]};
`;

const ModalContainer = styled.div<{ width?: string }>`
  background: ${colors.cardBg};
  border: 1px solid ${colors.border};
  border-radius: ${radius.xl};
  box-shadow: ${shadow.xl};
  width: 100%;
  max-width: ${({ width }) => width || '500px'};
  max-height: calc(100vh - 48px);
  display: flex;
  flex-direction: column;
`;

const ModalContent = styled.div`
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  min-height: 0;

  &::-webkit-scrollbar {
    width: 8px;
  }

  &::-webkit-scrollbar-track {
    background: transparent;
  }

  &::-webkit-scrollbar-thumb {
    background: ${colors.border};
    border-radius: 4px;
  }

  &::-webkit-scrollbar-thumb:hover {
    background: ${colors.borderLight};
  }
`;

const ModalHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: ${spacing[5]} ${spacing[6]};
  border-bottom: 1px solid ${colors.border};
`;

const ModalTitle = styled.h2`
  font-size: 18px;
  font-weight: 500;
  color: ${colors.foreground};
`;

const CloseButton = styled.button`
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: ${radius.md};
  color: ${colors.mutedForeground};
  transition: all 0.15s ease;

  &:hover {
    background: ${colors.hoverBg};
    color: ${colors.foreground};
  }
`;

const ModalBody = styled.div`
  padding: ${spacing[6]};
`;

const ModalFooter = styled.div`
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: ${spacing[3]};
  padding: ${spacing[4]} ${spacing[6]};
  border-top: 1px solid ${colors.border};
`;

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
  width?: string;
}

export function Modal({ open, onClose, title, children, footer, width }: ModalProps) {
  useEffect(() => {
    if (open) {
      document.body.style.overflow = 'hidden';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [open]);

  if (!open) return null;

  return (
    <Overlay>
      <ModalContainer width={width}>
        <ModalHeader>
          <ModalTitle>{title}</ModalTitle>
          <CloseButton onClick={onClose} aria-label="Close">
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
            >
              <path d="M18 6L6 18M6 6l12 12" />
            </svg>
          </CloseButton>
        </ModalHeader>
        <ModalContent>
          <ModalBody>{children}</ModalBody>
        </ModalContent>
        {footer && <ModalFooter>{footer}</ModalFooter>}
      </ModalContainer>
    </Overlay>
  );
}

export { ModalFooter };
