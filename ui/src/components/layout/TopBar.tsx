import { useLocation } from 'react-router-dom';
import styled from '@emotion/styled';
import { colors, fontSize, radius, spacing } from '../../styles';
import { useTheme } from '../../contexts/ThemeContext';

const Bar = styled.header`
  display: flex;
  align-items: center;
  gap: ${spacing[2]};
  padding: ${spacing[2]} ${spacing[4]};
  border-bottom: 1px solid ${colors.border};
`;

const IconButton = styled.button`
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: ${radius.md};
  color: ${colors.mutedForeground};
  background: transparent;
  border: none;
  cursor: pointer;
  transition: opacity 0.15s ease;
  opacity: 0.6;
  flex-shrink: 0;

  &:hover {
    opacity: 1;
    color: ${colors.foreground};
  }

  svg {
    width: 16px;
    height: 16px;
  }
`;

const Separator = styled.span`
  color: ${colors.border};
  font-size: ${fontSize.sm};
  user-select: none;
  flex-shrink: 0;
`;

const PageTitle = styled.span`
  font-size: ${fontSize.sm};
  font-weight: 500;
  color: ${colors.foreground};
  white-space: nowrap;
`;

const Spacer = styled.div`
  flex: 1;
`;

const PAGE_TITLES: Record<string, string> = {
  '/budgets': 'Budgets',
  '/rate-limits': 'Rate Limits',
  '/approvals': 'Approvals',
  '/alerts': 'Alerts',
  '/audit': 'Audit',
  '/model-costs': 'Model Costs',
};

function getPageTitle(pathname: string): string {
  if (PAGE_TITLES[pathname]) return PAGE_TITLES[pathname];
  if (pathname.startsWith('/budgets/')) return 'Budget Detail';
  if (pathname.startsWith('/rate-limits/')) return 'Rate Limit Detail';
  return '';
}

interface TopBarProps {
  onToggleSidebar: () => void;
}

export function TopBar({ onToggleSidebar }: TopBarProps) {
  const { theme, toggleTheme } = useTheme();
  const location = useLocation();
  const title = getPageTitle(location.pathname);

  return (
    <Bar>
      <IconButton onClick={onToggleSidebar} title="Toggle sidebar">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <rect width="18" height="18" x="3" y="3" rx="2" ry="2" />
          <line x1="9" x2="9" y1="3" y2="21" />
        </svg>
      </IconButton>
      {title && (
        <>
          <Separator>|</Separator>
          <PageTitle>{title}</PageTitle>
        </>
      )}
      <Spacer />
      <IconButton
        onClick={toggleTheme}
        title={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
      >
        {theme === 'dark' ? (
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
          </svg>
        ) : (
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="5" />
            <line x1="12" y1="1" x2="12" y2="3" />
            <line x1="12" y1="21" x2="12" y2="23" />
            <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
            <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
            <line x1="1" y1="12" x2="3" y2="12" />
            <line x1="21" y1="12" x2="23" y2="12" />
            <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
            <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
          </svg>
        )}
      </IconButton>
    </Bar>
  );
}
