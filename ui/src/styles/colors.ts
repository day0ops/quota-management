// CSS variable-based colors for theming
export const colors = {
  // Page backgrounds
  background: 'var(--color-background)',
  cardBg: 'var(--color-card-bg)',
  surfaceBg: 'var(--color-surface-bg)',

  // Text colors
  foreground: 'var(--color-foreground)',
  mutedForeground: 'var(--color-muted-foreground)',
  dimForeground: 'var(--color-dim-foreground)',

  // Borders
  border: 'var(--color-border)',
  borderLight: 'var(--color-border-light)',
  borderDark: 'var(--color-border-dark)',

  // Primary action (purple — matches kagent-enterprise brandPrimary500)
  primary: '#8023C3',
  primaryHover: '#9333D6',
  primaryActive: '#6D1EA3',

  // Secondary (gray)
  secondary: 'var(--color-secondary)',
  secondaryHover: 'var(--color-secondary-hover)',
  secondaryActive: 'var(--color-secondary-active)',

  // Semantic colors
  error: 'var(--color-error)',
  errorBg: 'var(--color-error-bg)',
  errorBorder: 'var(--color-error-border)',

  success: 'var(--color-success)',
  successBg: 'var(--color-success-bg)',
  successBorder: 'var(--color-success-border)',

  warning: 'var(--color-warning)',
  warningBg: 'var(--color-warning-bg)',
  warningBorder: 'var(--color-warning-border)',

  info: 'var(--color-info)',
  infoBg: 'var(--color-info-bg)',
  infoBorder: 'var(--color-info-border)',

  // Interactive states
  hoverBg: 'var(--color-hover-bg)',
  activeBg: 'var(--color-active-bg)',

  // Table
  tableRowHover: 'var(--color-table-row-hover)',

  // Sidebar
  sidebarBg: 'var(--color-sidebar-bg)',
  sidebarBorder: 'var(--color-sidebar-border)',
  sidebarItemHover: 'var(--color-sidebar-item-hover)',
  sidebarItemActive: 'var(--color-sidebar-item-active)',
  sidebarNavHover: 'var(--color-sidebar-nav-hover)',
  sidebarNavActive: 'var(--color-sidebar-nav-active)',
  sidebarText: 'var(--color-sidebar-text)',
  sidebarMuted: 'var(--color-sidebar-muted)',
  sidebarSurface: 'var(--color-sidebar-surface)',
  sidebarSurfaceHover: 'var(--color-sidebar-surface-hover)',
  sidebarBorderGradient: 'var(--color-sidebar-border-gradient)',
} as const;

export type ColorKey = keyof typeof colors;

// Dark theme values
export const darkTheme = {
  '--color-background': '#0D0E15',
  '--color-card-bg': '#11101C',
  '--color-surface-bg': '#12101C',
  '--color-foreground': '#FAFAFA',
  '--color-muted-foreground': '#A1A1AA',
  '--color-dim-foreground': '#575961',
  '--color-border': '#27242E',
  '--color-border-light': '#3B3C46',
  '--color-border-dark': '#1E1E25',
  '--color-secondary': '#27272A',
  '--color-secondary-hover': '#34343B',
  '--color-secondary-active': '#3F3F46',
  '--color-error': '#F87171',
  '--color-error-bg': '#450A0A',
  '--color-error-border': '#7F1D1D',
  '--color-success': '#4ADE80',
  '--color-success-bg': '#052E16',
  '--color-success-border': '#14532D',
  '--color-warning': '#FB923C',
  '--color-warning-bg': '#431407',
  '--color-warning-border': '#7C2D12',
  '--color-info': '#60A5FA',
  '--color-info-bg': '#172554',
  '--color-info-border': '#1E3A8A',
  '--color-hover-bg': '#1C1C26',
  '--color-active-bg': '#262736',
  '--color-table-row-hover': '#1C1C26',
  '--color-sidebar-bg': '#0E0E17',
  '--color-sidebar-border': '#27242E',
  '--color-sidebar-item-hover': 'rgba(112, 76, 255, 0.06)',
  '--color-sidebar-item-active': 'rgba(112, 76, 255, 0.14)',
  '--color-sidebar-nav-hover': 'linear-gradient(120deg, #1b1641, rgba(29, 22, 62, 0.7))',
  '--color-sidebar-nav-active': 'linear-gradient(120deg, #231D50, #1D163E)',
  '--color-sidebar-text': '#FAFAFA',
  '--color-sidebar-muted': '#A1A1AA',
  '--color-sidebar-surface': 'rgba(255, 255, 255, 0.04)',
  '--color-sidebar-surface-hover': 'rgba(255, 255, 255, 0.07)',
  '--color-sidebar-border-gradient':
    'radial-gradient(ellipse at right 60%, transparent 15%, #3d3d4d)',
  '--color-scheme': 'dark',
  '--color-tooltip-bg': '#d4d4d8',
  '--color-tooltip-text': '#18181b',
  '--color-chart-accent': '#6844ff',
  '--color-chart-actual': '#8b5cf6',
  '--color-chart-error': '#f87171',
};

// Light theme values
export const lightTheme = {
  '--color-background': '#F8FAFC',
  '--color-card-bg': '#FFFFFF',
  '--color-surface-bg': '#F1F5F9',
  '--color-foreground': '#0F172A',
  '--color-muted-foreground': '#64748B',
  '--color-dim-foreground': '#94A3B8',
  '--color-border': '#E2E8F0',
  '--color-border-light': '#CBD5E1',
  '--color-border-dark': '#E2E8F0',
  '--color-secondary': '#E2E8F0',
  '--color-secondary-hover': '#CBD5E1',
  '--color-secondary-active': '#94A3B8',
  '--color-error': '#991B1B',
  '--color-error-bg': '#FEF2F2',
  '--color-error-border': '#FCA5A5',
  '--color-success': '#166534',
  '--color-success-bg': '#F0FDF4',
  '--color-success-border': '#86EFAC',
  '--color-warning': '#92400E',
  '--color-warning-bg': '#FFFBEB',
  '--color-warning-border': '#FCD34D',
  '--color-info': '#2563EB',
  '--color-info-bg': '#EFF6FF',
  '--color-info-border': '#BFDBFE',
  '--color-hover-bg': '#F1F5F9',
  '--color-active-bg': '#E2E8F0',
  '--color-table-row-hover': '#F1F5F9',
  '--color-sidebar-bg': '#FFFFFF',
  '--color-sidebar-border': '#E2E8F0',
  '--color-sidebar-item-hover': 'rgba(128, 35, 195, 0.06)',
  '--color-sidebar-item-active': 'rgba(128, 35, 195, 0.12)',
  '--color-sidebar-nav-hover':
    'linear-gradient(120deg, rgba(128, 35, 195, 0.09), rgba(109, 40, 217, 0.06))',
  '--color-sidebar-nav-active':
    'linear-gradient(120deg, rgba(128, 35, 195, 0.16), rgba(109, 40, 217, 0.11))',
  '--color-sidebar-text': '#0F172A',
  '--color-sidebar-muted': '#64748B',
  '--color-sidebar-surface': 'rgba(0, 0, 0, 0.04)',
  '--color-sidebar-surface-hover': 'rgba(0, 0, 0, 0.07)',
  '--color-sidebar-border-gradient':
    'radial-gradient(ellipse at right 60%, transparent 15%, #CBD5E1)',
  '--color-scheme': 'light',
  '--color-tooltip-bg': '#FFFFFF',
  '--color-tooltip-text': '#18181b',
  '--color-chart-accent': '#4f46e5',
  '--color-chart-actual': '#6d28d9',
  '--color-chart-error': '#dc2626',
};
