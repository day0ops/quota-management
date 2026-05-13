import { ReactNode } from 'react';
import styled from '@emotion/styled';
import { colors, spacing } from '../../styles';
import { Modal } from './Modal';
import { Button } from './Button';

const Content = styled.div`
  text-align: center;
`;

const Message = styled.p`
  color: ${colors.mutedForeground};
  margin-bottom: ${spacing[4]};
`;

const ButtonGroup = styled.div`
  display: flex;
  gap: ${spacing[3]};
  justify-content: center;
  margin-top: ${spacing[5]};
`;

interface ConfirmDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: 'danger' | 'primary';
  loading?: boolean;
  hideCancel?: boolean;
}

export function ConfirmDialog({
  open,
  onClose,
  onConfirm,
  title,
  message,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  variant = 'danger',
  loading = false,
  hideCancel = false,
}: ConfirmDialogProps) {
  return (
    <Modal open={open} onClose={onClose} title={title} width="400px">
      <Content>
        <Message>{message}</Message>
        <ButtonGroup>
          {!hideCancel && (
            <Button variant="secondary" onClick={onClose} disabled={loading}>
              {cancelLabel}
            </Button>
          )}
          <Button variant={variant} onClick={onConfirm} disabled={loading}>
            {loading ? 'Loading...' : confirmLabel}
          </Button>
        </ButtonGroup>
      </Content>
    </Modal>
  );
}
