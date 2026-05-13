package models

// PaginationParams holds parsed pagination query parameters.
type PaginationParams struct {
	Page     int
	PageSize int
}

// PaginationMeta holds pagination metadata for responses.
type PaginationMeta struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalCount int `json:"total_count"`
	TotalPages int `json:"total_pages"`
}

// PaginatedResponse wraps data with pagination metadata.
type PaginatedResponse struct {
	Data       interface{}    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// Offset returns the SQL OFFSET for the current page.
func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// NewPaginationMeta creates pagination metadata from total count.
func NewPaginationMeta(params PaginationParams, totalCount int) PaginationMeta {
	totalPages := totalCount / params.PageSize
	if totalCount%params.PageSize > 0 {
		totalPages++
	}
	return PaginationMeta{
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}
}
