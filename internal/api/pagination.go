package api

import (
	"net/http"
	"strconv"

	"github.com/agentgateway/quota-management/internal/models"
)

const (
	defaultPageSize = 30
	maxPageSize     = 30
)

func parsePagination(r *http.Request) models.PaginationParams {
	page := 1
	pageSize := defaultPageSize

	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			if v > maxPageSize {
				v = maxPageSize
			}
			pageSize = v
		}
	}

	return models.PaginationParams{Page: page, PageSize: pageSize}
}

func writePaginatedJSON(w http.ResponseWriter, status int, data interface{}, params models.PaginationParams, totalCount int) {
	resp := models.PaginatedResponse{
		Data:       data,
		Pagination: models.NewPaginationMeta(params, totalCount),
	}
	writeJSON(w, status, resp)
}
