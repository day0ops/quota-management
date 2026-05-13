import { useState, useRef, useEffect } from 'react';
import { NavLink } from 'react-router-dom';
import styled from '@emotion/styled';
import { colors, radius, spacing, fontSize } from '../../styles';
import { useAuth } from '../../contexts/AuthContext';
import { approvalsApi } from '../../api/approvals';
import { budgetsApi } from '../../api/budgets';
import { rateLimitsApi } from '../../api/rate-limits';
import { alertsApi, useAlertCount } from '../../api/alerts';
import { useSWRApi, CacheKeys, invalidateKey } from '../../hooks/useSWR';
import { config } from '../../config';

const SidebarContainer = styled.aside<{ $collapsed: boolean }>`
  width: ${({ $collapsed }) => ($collapsed ? '72px' : '240px')};
  min-width: ${({ $collapsed }) => ($collapsed ? '72px' : '240px')};
  transition:
    width 0.25s ease,
    min-width 0.25s ease;
  background: var(--color-sidebar-bg);
  border-radius: 16px;
  display: flex;
  flex-direction: column;
  height: 100%;
  position: relative;
  align-self: stretch;

  &::after {
    content: '';
    pointer-events: none;
    position: absolute;
    inset: 0;
    border-radius: 16px;
    border: 1px solid transparent;
    background: var(--color-sidebar-border-gradient) border-box;
    mask:
      linear-gradient(#fff 0 0) padding-box,
      linear-gradient(#fff 0 0);
    mask-composite: exclude;
  }
`;

const Logo = styled.div<{ $collapsed: boolean }>`
  margin-top: 10px;
  padding: 8px ${spacing[3]} 6px ${({ $collapsed }) => ($collapsed ? spacing[3] : '28px')};
  border-bottom: 1px solid ${colors.sidebarBorder};
  min-height: 65px;
  display: flex;
  align-items: center;
  gap: ${spacing[2]};
  justify-content: ${({ $collapsed }) => ($collapsed ? 'center' : 'space-between')};
  position: relative;
`;

const LogoMark = styled.div`
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: linear-gradient(135deg, #8023c3, #6d28d9);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  font-size: 13px;
  font-weight: 700;
  color: white;
  letter-spacing: -0.5px;
`;

const LogoTextGroup = styled.div<{ $collapsed: boolean }>`
  flex: ${({ $collapsed }) => ($collapsed ? '0' : '1')};
  max-width: ${({ $collapsed }) => ($collapsed ? '0' : '200px')};
  overflow: hidden;
  opacity: ${({ $collapsed }) => ($collapsed ? 0 : 1)};
  transition:
    opacity 0.15s ease,
    max-width 0.25s ease,
    flex 0.25s ease;
  pointer-events: ${({ $collapsed }) => ($collapsed ? 'none' : 'auto')};
  white-space: nowrap;
`;

const LogoText = styled.h1`
  font-size: ${fontSize.md};
  font-weight: 600;
  color: ${colors.sidebarText};
  letter-spacing: -0.01em;
  white-space: nowrap;
`;

const LogoSubtext = styled.span`
  font-size: ${fontSize.xs};
  color: ${colors.sidebarMuted};
  display: block;
  margin-top: 2px;
  white-space: nowrap;
`;

const Nav = styled.nav<{ $collapsed: boolean }>`
  padding: ${spacing[5]} ${({ $collapsed }) => ($collapsed ? spacing[2] : spacing[5])} 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: ${spacing[5]};
`;

const NavSection = styled.div``;

const NavSectionTitle = styled.h3<{ $collapsed: boolean }>`
  font-size: 10px;
  font-weight: 500;
  color: ${colors.sidebarMuted};
  text-transform: uppercase;
  letter-spacing: 0.5px;
  padding: 0 ${spacing[2]};
  margin-bottom: ${({ $collapsed }) => ($collapsed ? 0 : spacing[1])};
  opacity: ${({ $collapsed }) => ($collapsed ? 0 : 1)};
  max-height: ${({ $collapsed }) => ($collapsed ? '0' : '24px')};
  overflow: hidden;
  transition:
    opacity 0.15s ease,
    max-height 0.25s ease,
    margin-bottom 0.25s ease;
  white-space: nowrap;
`;

const NavItem = styled(NavLink)<{ $collapsed: boolean }>`
  display: flex;
  align-items: center;
  gap: ${({ $collapsed }) => ($collapsed ? 0 : spacing[3])};
  padding: 10px ${({ $collapsed }) => ($collapsed ? '12px' : spacing[4])};
  border-radius: 9999px;
  font-size: ${fontSize.sm};
  font-weight: 400;
  color: ${colors.sidebarText};
  transition: all 0.15s ease;
  position: relative;
  justify-content: ${({ $collapsed }) => ($collapsed ? 'center' : 'flex-start')};

  &:hover {
    background: var(--color-sidebar-nav-hover);
  }

  &.active {
    background: var(--color-sidebar-nav-active);
    font-weight: 500;

    &::before {
      content: '';
      position: absolute;
      inset: 0;
      border-radius: 9999px;
      padding: 1px;
      background: linear-gradient(160deg, rgba(255, 255, 255, 0.22) 0%, transparent 70%);
      -webkit-mask:
        linear-gradient(#fff 0 0) content-box,
        linear-gradient(#fff 0 0);
      -webkit-mask-composite: xor;
      mask-composite: exclude;
      pointer-events: none;
    }
  }

  svg {
    width: 16px;
    height: 16px;
    flex-shrink: 0;
  }

  svg path {
    opacity: 0.7;
    transition: opacity 0.08s ease-out;
  }

  &.active svg path {
    opacity: 1;
  }
`;

const NavLabel = styled.span<{ $collapsed: boolean }>`
  opacity: ${({ $collapsed }) => ($collapsed ? 0 : 1)};
  max-width: ${({ $collapsed }) => ($collapsed ? '0' : '160px')};
  overflow: hidden;
  white-space: nowrap;
  transition:
    opacity 0.15s ease,
    max-width 0.25s ease;
`;

const NavBadge = styled.span<{ $collapsed: boolean }>`
  background: ${colors.error};
  color: white;
  font-size: 11px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: ${radius.full};
  margin-left: auto;
  min-width: 18px;
  text-align: center;
  display: ${({ $collapsed }) => ($collapsed ? 'none' : 'inline-block')};
`;

const SummaryStats = styled.div<{ $collapsed: boolean }>`
  padding: ${({ $collapsed }) =>
    $collapsed ? `${spacing[2]} ${spacing[2]}` : `${spacing[3]} ${spacing[4]}`};
  border-top: 1px solid ${colors.sidebarBorder};
  margin-top: auto;
  transition: padding 0.25s ease;
`;

const SummaryTitle = styled.div<{ $collapsed: boolean }>`
  font-size: 10px;
  font-weight: 500;
  color: ${colors.sidebarMuted};
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: ${spacing[2]};
  opacity: ${({ $collapsed }) => ($collapsed ? 0 : 1)};
  max-height: ${({ $collapsed }) => ($collapsed ? '0' : '20px')};
  overflow: hidden;
  transition:
    opacity 0.15s ease,
    max-height 0.25s ease,
    margin-bottom 0.25s ease;
  ${({ $collapsed }) => $collapsed && 'margin-bottom: 0;'}
`;

const StatsRow = styled.div<{ $collapsed: boolean }>`
  display: flex;
  flex-direction: ${({ $collapsed }) => ($collapsed ? 'column' : 'row')};
  gap: ${spacing[2]};
`;

const StatBox = styled.div`
  flex: 1;
  text-align: center;
  padding: ${spacing[2]};
  border-radius: ${radius.md};
  background: ${colors.sidebarSurface};
`;

const StatValue = styled.div`
  font-size: ${fontSize.lg};
  font-weight: 600;
  color: ${colors.sidebarText};
`;

const StatLabel = styled.div<{ $collapsed: boolean }>`
  font-size: 10px;
  color: ${colors.sidebarMuted};
  text-transform: uppercase;
  letter-spacing: 0.05em;
  opacity: ${({ $collapsed }) => ($collapsed ? 0 : 1)};
  max-height: ${({ $collapsed }) => ($collapsed ? '0' : '16px')};
  overflow: hidden;
  transition:
    opacity 0.15s ease,
    max-height 0.25s ease;
`;

const StatAbbr = styled.div<{ $collapsed: boolean }>`
  font-size: 10px;
  font-weight: 600;
  color: ${colors.sidebarMuted};
  text-transform: uppercase;
  letter-spacing: 0.05em;
  opacity: ${({ $collapsed }) => ($collapsed ? 1 : 0)};
  max-height: ${({ $collapsed }) => ($collapsed ? '16px' : '0')};
  overflow: hidden;
  transition:
    opacity 0.15s ease,
    max-height 0.25s ease;
`;

const UserSection = styled.div<{ $collapsed: boolean }>`
  padding: ${({ $collapsed }) => ($collapsed ? spacing[2] : spacing[4])};
  border-top: 1px solid ${colors.sidebarBorder};
  position: relative;
  transition: padding 0.25s ease;
`;

const UserInfo = styled.button`
  display: flex;
  align-items: center;
  gap: ${spacing[3]};
  padding: ${spacing[2]} ${spacing[3]};
  border-radius: 9999px;
  background: ${colors.sidebarSurface};
  width: 100%;
  border: none;
  cursor: pointer;
  transition: background 0.15s ease;

  &:hover {
    background: ${colors.sidebarSurfaceHover};
  }
`;

const LogoutFlyout = styled.div<{ $open: boolean }>`
  position: absolute;
  left: calc(100% + ${spacing[3]});
  top: 50%;
  transform: translateY(-50%) translateX(${({ $open }) => ($open ? '0' : '-6px')});
  z-index: 100;
  opacity: ${({ $open }) => ($open ? 1 : 0)};
  visibility: ${({ $open }) => ($open ? 'visible' : 'hidden')};
  transition: all 0.15s ease;
`;

const LogoutPill = styled.button`
  display: flex;
  align-items: center;
  gap: ${spacing[2]};
  padding: 8px 14px;
  border-radius: 9999px;
  background: linear-gradient(117deg, #6844ff -23.54%, #1d283a 223.49%);
  border: none;
  cursor: pointer;
  font-size: ${fontSize.sm};
  font-weight: 500;
  color: #fafafa;
  white-space: nowrap;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
  position: relative;
  transition: all 0.15s ease;

  &::before {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: 9999px;
    padding: 1px;
    background: linear-gradient(160deg, #3b3c46 10%, #0b0c17);
    -webkit-mask:
      linear-gradient(#fff 0 0) content-box,
      linear-gradient(#fff 0 0);
    -webkit-mask-composite: xor;
    mask-composite: exclude;
    pointer-events: none;
  }

  &:hover {
    background: linear-gradient(117deg, #382b93 -23.54%, #2b2568 223.49%);
  }

  &:active {
    background: linear-gradient(117deg, #31267a -23.54%, #282658 223.49%);
  }

  svg {
    width: 14px;
    height: 14px;
    flex-shrink: 0;
  }
`;

const UserAvatar = styled.div`
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: linear-gradient(135deg, #6844ff, #8023c3);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: ${fontSize.sm};
  font-weight: 600;
  color: #ffffff;
  flex-shrink: 0;
`;

const UserDetails = styled.div<{ $collapsed: boolean }>`
  flex: ${({ $collapsed }) => ($collapsed ? '0' : '1')};
  max-width: ${({ $collapsed }) => ($collapsed ? '0' : '160px')};
  overflow: hidden;
  opacity: ${({ $collapsed }) => ($collapsed ? 0 : 1)};
  transition:
    opacity 0.15s ease,
    max-width 0.25s ease,
    flex 0.25s ease;
  white-space: nowrap;
`;

const UserEmail = styled.div`
  font-size: ${fontSize.sm};
  color: ${colors.sidebarText};
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  text-align: left;
`;

const UserRole = styled.div`
  font-size: ${fontSize.xs};
  color: ${colors.sidebarMuted};
  text-align: left;
`;

const NotAuthenticated = styled.div`
  padding: ${spacing[2]} ${spacing[3]};
  font-size: ${fontSize.xs};
  color: ${colors.sidebarMuted};
  text-align: center;
`;

function formatCompact(num: number | null): string {
  if (num === null) return '-';
  if (num < 1000) return num.toString();
  if (num < 10000) return (num / 1000).toFixed(1).replace(/\.0$/, '') + 'K';
  if (num < 1000000) return Math.round(num / 1000) + 'K';
  return (num / 1000000).toFixed(1).replace(/\.0$/, '') + 'M';
}

async function fetchSidebarStats() {
  const promises: Promise<any>[] = [
    approvalsApi.count(),
    budgetsApi.list(1, 1, { enabledOnly: true }),
    alertsApi.count(),
  ];
  if (config.enableRateLimits) {
    promises.push(rateLimitsApi.count());
    promises.push(rateLimitsApi.list(1, 1, { enabledOnly: true }));
  }
  const [budgetApprovals, budgets, alertCount, rateLimitApprovals, rateLimits] =
    await Promise.all(promises);
  return {
    pendingCount:
      (budgetApprovals.count ?? 0) +
      (config.enableRateLimits ? (rateLimitApprovals?.count ?? 0) : 0),
    budgetCount: budgets.pagination?.total_count ?? 0,
    rateLimitCount: config.enableRateLimits ? (rateLimits?.pagination?.total_count ?? 0) : null,
    alertCount: alertCount?.active ?? 0,
  };
}

interface SidebarProps {
  collapsed: boolean;
}

export function Sidebar({ collapsed }: SidebarProps) {
  const { identity, loading, permissions } = useAuth();
  const [flyoutOpen, setFlyoutOpen] = useState(false);
  const userMenuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (userMenuRef.current && !userMenuRef.current.contains(event.target as Node)) {
        setFlyoutOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const { data: stats } = useSWRApi(CacheKeys.sidebarStats, fetchSidebarStats, {
    refreshInterval: 15000,
    revalidateOnFocus: true,
  });

  // Invalidate sidebar stats when alert count changes so badge stays in sync.
  const { data: liveAlertCount } = useAlertCount();
  useEffect(() => {
    invalidateKey(CacheKeys.sidebarStats);
  }, [liveAlertCount?.active]);

  const pendingCount = stats?.pendingCount ?? 0;
  const budgetCount = stats?.budgetCount ?? null;
  const rateLimitCount = stats?.rateLimitCount ?? null;
  const alertCount = stats?.alertCount ?? 0;

  const handleLogout = () => {
    window.location.href = '/logout';
  };

  return (
    <SidebarContainer $collapsed={collapsed}>
      <Logo $collapsed={collapsed}>
        <LogoMark>QM</LogoMark>
        <LogoTextGroup $collapsed={collapsed}>
          <LogoText>Quota Management</LogoText>
          <LogoSubtext>Management Console</LogoSubtext>
        </LogoTextGroup>
      </Logo>

      <Nav $collapsed={collapsed}>
        <NavSection>
          <NavItem $collapsed={collapsed} to="/budgets" title={collapsed ? 'Budgets' : undefined}>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <rect x="2" y="3" width="20" height="14" rx="2" ry="2" />
              <path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16" />
            </svg>
            <NavLabel $collapsed={collapsed}>Budgets</NavLabel>
          </NavItem>
          {config.enableRateLimits && (
            <NavItem
              $collapsed={collapsed}
              to="/rate-limits"
              title={collapsed ? 'Rate Limits' : undefined}
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83" />
              </svg>
              <NavLabel $collapsed={collapsed}>Rate Limits</NavLabel>
            </NavItem>
          )}
          {permissions.isOrgAdmin && (
            <NavItem
              $collapsed={collapsed}
              to="/approvals"
              title={
                collapsed ? `Approvals${pendingCount > 0 ? ` (${pendingCount})` : ''}` : undefined
              }
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                <polyline points="22 4 12 14.01 9 11.01" />
              </svg>
              <NavLabel $collapsed={collapsed}>Approvals</NavLabel>
              {pendingCount > 0 && <NavBadge $collapsed={collapsed}>{pendingCount}</NavBadge>}
            </NavItem>
          )}
          <NavItem
            $collapsed={collapsed}
            to="/alerts"
            title={collapsed ? `Alerts${alertCount > 0 ? ` (${alertCount})` : ''}` : undefined}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
              <path d="M13.73 21a2 2 0 0 1-3.46 0" />
            </svg>
            <NavLabel $collapsed={collapsed}>Alerts</NavLabel>
            {alertCount > 0 && <NavBadge $collapsed={collapsed}>{alertCount}</NavBadge>}
          </NavItem>
          <NavItem $collapsed={collapsed} to="/audit" title={collapsed ? 'Audit' : undefined}>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2" />
              <rect x="8" y="2" width="8" height="4" rx="1" ry="1" />
              <line x1="8" y1="12" x2="16" y2="12" />
              <line x1="8" y1="16" x2="12" y2="16" />
            </svg>
            <NavLabel $collapsed={collapsed}>Audit</NavLabel>
          </NavItem>
        </NavSection>
        <NavSection>
          <NavSectionTitle $collapsed={collapsed}>Settings</NavSectionTitle>
          <NavItem
            $collapsed={collapsed}
            to="/model-costs"
            title={collapsed ? 'Model Costs' : undefined}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M12 2L2 7l10 5 10-5-10-5z" />
              <path d="M2 17l10 5 10-5" />
              <path d="M2 12l10 5 10-5" />
            </svg>
            <NavLabel $collapsed={collapsed}>Model Costs</NavLabel>
          </NavItem>
        </NavSection>
      </Nav>

      <SummaryStats $collapsed={collapsed}>
        <SummaryTitle $collapsed={collapsed}>At a Glance</SummaryTitle>
        <StatsRow $collapsed={collapsed}>
          <StatBox>
            <StatAbbr $collapsed={collapsed}>B</StatAbbr>
            <StatValue>{formatCompact(budgetCount)}</StatValue>
            <StatLabel $collapsed={collapsed}>Budgets</StatLabel>
          </StatBox>
          {config.enableRateLimits && (
            <StatBox>
              <StatAbbr $collapsed={collapsed}>L</StatAbbr>
              <StatValue>{formatCompact(rateLimitCount)}</StatValue>
              <StatLabel $collapsed={collapsed}>Limits</StatLabel>
            </StatBox>
          )}
          {permissions.isOrgAdmin && (
            <StatBox>
              <StatAbbr $collapsed={collapsed}>P</StatAbbr>
              <StatValue>{formatCompact(pendingCount)}</StatValue>
              <StatLabel $collapsed={collapsed}>Pending</StatLabel>
            </StatBox>
          )}
        </StatsRow>
      </SummaryStats>

      <UserSection $collapsed={collapsed} ref={userMenuRef}>
        {loading ? (
          <NotAuthenticated>Loading...</NotAuthenticated>
        ) : identity?.authenticated ? (
          <>
            <UserInfo onClick={() => setFlyoutOpen(prev => !prev)}>
              <UserAvatar>{identity.email ? identity.email[0].toUpperCase() : 'U'}</UserAvatar>
              <UserDetails $collapsed={collapsed}>
                <UserEmail>{identity.email || identity.subject || 'User'}</UserEmail>
                <UserRole>
                  {identity.is_org
                    ? `Admin • ${identity.org_id}`
                    : identity.team_id
                      ? `Member • ${identity.team_id}`
                      : 'Guest'}
                </UserRole>
              </UserDetails>
            </UserInfo>
            <LogoutFlyout $open={flyoutOpen}>
              <LogoutPill onClick={handleLogout}>
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
                  <polyline points="16 17 21 12 16 7" />
                  <line x1="21" y1="12" x2="9" y2="12" />
                </svg>
                Logout
              </LogoutPill>
            </LogoutFlyout>
          </>
        ) : (
          <NotAuthenticated>Not authenticated</NotAuthenticated>
        )}
      </UserSection>
    </SidebarContainer>
  );
}
