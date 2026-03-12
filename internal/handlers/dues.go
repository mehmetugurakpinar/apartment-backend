package handlers

import (
	"strconv"
	"time"

	"apartment-backend/internal/middleware"
	"apartment-backend/internal/models"
	"apartment-backend/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type DuesHandler struct {
	financialRepo *repository.FinancialRepository
}

func NewDuesHandler(financialRepo *repository.FinancialRepository) *DuesHandler {
	return &DuesHandler{financialRepo: financialRepo}
}

func (h *DuesHandler) GetDues(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	plans, err := h.financialRepo.GetDuesPlans(c.Context(), buildingID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(plans, ""))
}

func (h *DuesHandler) CreateDues(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	var req models.CreateDuesPlanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid due_date format (YYYY-MM-DD)"))
	}

	userID := middleware.GetUserID(c)
	plan := &models.DuesPlan{
		BuildingID:  buildingID,
		Title:       req.Title,
		Amount:      req.Amount,
		PeriodMonth: req.PeriodMonth,
		PeriodYear:  req.PeriodYear,
		DueDate:     dueDate,
		CreatedBy:   &userID,
	}

	if err := h.financialRepo.CreateDuesPlan(c.Context(), plan); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(plan, "Dues plan created"))
}

func (h *DuesHandler) PayDues(c *fiber.Ctx) error {
	planID, err := uuid.Parse(c.Params("planId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid plan ID"))
	}

	var req models.PayDuesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	unitID, err := uuid.Parse(req.UnitID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid unit_id"))
	}

	payment := &models.DuePayment{
		DuesPlanID: planID,
		UnitID:     unitID,
		PaidAmount: &req.PaidAmount,
		Notes:      req.Notes,
	}

	if err := h.financialRepo.CreatePayment(c.Context(), payment); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(payment, "Payment recorded"))
}

func (h *DuesHandler) GetReport(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	month, _ := strconv.Atoi(c.Query("month", strconv.Itoa(int(time.Now().Month()))))
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(time.Now().Year())))

	report, err := h.financialRepo.GetDuesReport(c.Context(), buildingID, month, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(report, ""))
}

func (h *DuesHandler) UpdateDues(c *fiber.Ctx) error {
	planID, err := uuid.Parse(c.Params("planId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid plan ID"))
	}

	var req models.UpdateDuesPlanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	plan, err := h.financialRepo.UpdateDuesPlan(c.Context(), planID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(plan, "Dues plan updated"))
}

func (h *DuesHandler) DeleteDues(c *fiber.Ctx) error {
	planID, err := uuid.Parse(c.Params("planId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid plan ID"))
	}

	if err := h.financialRepo.DeleteDuesPlan(c.Context(), planID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(nil, "Dues plan deleted"))
}

func (h *DuesHandler) UpdateExpense(c *fiber.Ctx) error {
	expenseID, err := uuid.Parse(c.Params("expenseId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid expense ID"))
	}

	var req models.UpdateExpenseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	expense, err := h.financialRepo.UpdateExpense(c.Context(), expenseID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(expense, "Expense updated"))
}

func (h *DuesHandler) DeleteExpense(c *fiber.Ctx) error {
	expenseID, err := uuid.Parse(c.Params("expenseId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid expense ID"))
	}

	if err := h.financialRepo.DeleteExpense(c.Context(), expenseID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(nil, "Expense deleted"))
}

func (h *DuesHandler) GetExpenses(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	var pq models.PaginationQuery
	c.QueryParser(&pq)
	pq.SetDefaults()

	expenses, total, err := h.financialRepo.GetExpenses(c.Context(), buildingID, pq.Page, pq.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(models.NewPaginatedResponse(expenses, pq.Page, pq.Limit, total), ""))
}

func (h *DuesHandler) CreateExpense(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	var req models.CreateExpenseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid date format (YYYY-MM-DD)"))
	}

	userID := middleware.GetUserID(c)
	expense := &models.Expense{
		BuildingID:  buildingID,
		Category:    req.Category,
		Amount:      req.Amount,
		Description: req.Description,
		Date:        date,
		CreatedBy:   &userID,
	}

	if err := h.financialRepo.CreateExpense(c.Context(), expense); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(expense, "Expense recorded"))
}
