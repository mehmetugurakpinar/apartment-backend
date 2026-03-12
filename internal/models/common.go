package models

type PaginationQuery struct {
	Page  int `query:"page"`
	Limit int `query:"limit"`
}

func (p *PaginationQuery) SetDefaults() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 || p.Limit > 100 {
		p.Limit = 20
	}
}

func (p *PaginationQuery) Offset() int {
	return (p.Page - 1) * p.Limit
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	Total      int64       `json:"total"`
	TotalPages int         `json:"total_pages"`
}

func NewPaginatedResponse(data interface{}, page, limit int, total int64) PaginatedResponse {
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}
	return PaginatedResponse{
		Data:       data,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func SuccessResponse(data interface{}, message string) APIResponse {
	return APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

func ErrorResponse(err string) APIResponse {
	return APIResponse{
		Success: false,
		Error:   err,
	}
}
