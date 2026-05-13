import { useState } from 'react';
import styled from '@emotion/styled';
import { colors, spacing } from '../../styles';
import { Sidebar } from './Sidebar';
import { TopBar } from './TopBar';
import { Footer } from './Footer';

const SIDEBAR_COLLAPSED_KEY = 'sidebar-collapsed';

const LayoutContainer = styled.div`
  display: flex;
  min-height: 100vh;
  width: 100%;
  padding: ${spacing[3]};
  gap: ${spacing[3]};
  background: ${colors.background};
`;

const MainContent = styled.main`
  flex: 1;
  display: flex;
  flex-direction: column;
  background: ${colors.background};
  overflow-x: hidden;
  min-height: 0;
`;

const ContentArea = styled.div`
  flex: 1;
  padding: ${spacing[8]};
`;

interface AppLayoutProps {
  children: React.ReactNode;
}

export function AppLayout({ children }: AppLayoutProps) {
  const [collapsed, setCollapsed] = useState(
    () => localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === 'true'
  );

  const toggleCollapsed = () => {
    setCollapsed(prev => {
      const next = !prev;
      localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(next));
      return next;
    });
  };

  return (
    <LayoutContainer>
      <Sidebar collapsed={collapsed} />
      <MainContent>
        <TopBar onToggleSidebar={toggleCollapsed} />
        <ContentArea>{children}</ContentArea>
        <Footer />
      </MainContent>
    </LayoutContainer>
  );
}
