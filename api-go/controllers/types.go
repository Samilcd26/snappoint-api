package controllers

type StandardResponse struct {
	Success    bool           `json:"success"`
	Data       interface{}    `json:"data,omitempty"`
	Meta       interface{}    `json:"meta,omitempty"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
	Message    string         `json:"message,omitempty"`
}

type PaginationMeta struct {
	CurrentPage int `json:"currentPage"`
	PageSize    int `json:"pageSize"`
	TotalItems  int64 `json:"totalItems"`
	TotalPages  int `json:"totalPages"`
} 