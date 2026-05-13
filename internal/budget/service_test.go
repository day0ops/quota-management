package budget

import (
	"testing"

	"github.com/agentgateway/quota-management/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestSortBudgetsByHierarchy_UserBeforeTeamBeforeOrg(t *testing.T) {
	budgets := []models.BudgetDefinition{
		{Name: "org-budget", EntityType: models.EntityTypeOrg},
		{Name: "user-budget", EntityType: models.EntityTypeUser},
		{Name: "team-budget", EntityType: models.EntityTypeTeam},
	}

	sorted := sortBudgetsByHierarchy(budgets)

	assert.Equal(t, 3, len(sorted))
	assert.Equal(t, models.EntityTypeUser, sorted[0].EntityType)
	assert.Equal(t, models.EntityTypeTeam, sorted[1].EntityType)
	assert.Equal(t, models.EntityTypeOrg, sorted[2].EntityType)
}

func TestSortBudgetsByHierarchy_NoProvider(t *testing.T) {
	budgets := []models.BudgetDefinition{
		{Name: "org-budget", EntityType: models.EntityTypeOrg},
	}
	sorted := sortBudgetsByHierarchy(budgets)
	assert.Equal(t, 1, len(sorted))
	assert.Equal(t, models.EntityTypeOrg, sorted[0].EntityType)
}
