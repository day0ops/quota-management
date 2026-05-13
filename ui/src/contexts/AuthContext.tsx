import { createContext, useContext, useEffect, useState, ReactNode, useMemo } from 'react';
import { Identity, BudgetDefinition } from '../api/types';
import { authApi } from '../api/auth';

interface Permissions {
  isOrgAdmin: boolean;
  isTeamMember: boolean;
  canCreateBudgets: boolean;
  canCreateOrgBudgets: boolean;
  canDeleteBudgets: boolean;
  canManageModelCosts: boolean;
  canToggleBudgetEnabled: boolean;
  canEditBudget: (budget: BudgetDefinition) => boolean;
  canViewBudget: (budget: BudgetDefinition) => boolean;
}

interface AuthContextType {
  identity: Identity | null;
  loading: boolean;
  permissions: Permissions;
  refresh: () => Promise<void>;
}

const defaultPermissions: Permissions = {
  isOrgAdmin: false,
  isTeamMember: false,
  canCreateBudgets: false,
  canCreateOrgBudgets: false,
  canDeleteBudgets: false,
  canManageModelCosts: false,
  canToggleBudgetEnabled: false,
  canEditBudget: () => false,
  canViewBudget: () => false,
};

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [identity, setIdentity] = useState<Identity | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchIdentity = async () => {
    try {
      const data = await authApi.getIdentity();
      setIdentity(data);
    } catch {
      setIdentity({ authenticated: false });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchIdentity();
  }, []);

  const refresh = async () => {
    setLoading(true);
    await fetchIdentity();
  };

  const permissions = useMemo<Permissions>(() => {
    if (!identity?.authenticated) {
      return defaultPermissions;
    }

    const isOrgAdmin = !!identity.is_org && !!identity.org_id;
    const isTeamMember = !identity.is_org && !!identity.team_id;

    return {
      isOrgAdmin,
      isTeamMember,
      // Both can create budgets (provider + team), but only org admins can create org budgets
      canCreateBudgets: isOrgAdmin || isTeamMember,
      canCreateOrgBudgets: isOrgAdmin,
      // Only org admins can delete budgets
      canDeleteBudgets: isOrgAdmin,
      // Only org admins can manage model costs
      canManageModelCosts: isOrgAdmin,
      // Only org admins can enable/disable budgets
      canToggleBudgetEnabled: isOrgAdmin,
      // Org admins can edit any budget in their org; team members only their team's
      // Budgets with no owner can be edited by org admins only
      // Team members cannot edit disabled budgets (org admin disabled it)
      canEditBudget: (budget: BudgetDefinition) => {
        // No owner = only org admins can edit
        if (!budget.owner_org_id && !budget.owner_team_id) {
          return isOrgAdmin;
        }
        if (isOrgAdmin) {
          return budget.owner_org_id === identity.org_id;
        }
        if (isTeamMember) {
          // Team members cannot edit disabled budgets
          if (!budget.enabled) {
            return false;
          }
          return budget.owner_team_id === identity.team_id;
        }
        return false;
      },
      // Org admins can view all budgets in their org
      // Team members can view their team's budgets AND their org's budgets (for parent selection in forms)
      // Budgets with no owner are visible to everyone (backwards compatibility)
      canViewBudget: (budget: BudgetDefinition) => {
        // No owner = visible to all
        if (!budget.owner_org_id && !budget.owner_team_id) {
          return true;
        }
        if (isOrgAdmin) {
          return budget.owner_org_id === identity.org_id;
        }
        if (isTeamMember) {
          // Team members can see their own team's budgets
          if (budget.owner_team_id === identity.team_id) {
            return true;
          }
          // Team members can also see org-level budgets from their org (for parent selection)
          if (budget.owner_org_id === identity.org_id && !budget.owner_team_id) {
            return true;
          }
        }
        return false;
      },
    };
  }, [identity]);

  return (
    <AuthContext.Provider value={{ identity, loading, permissions, refresh }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
