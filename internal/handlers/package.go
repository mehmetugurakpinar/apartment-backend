package handlers

import (
	"apartment-backend/internal/middleware"
	"apartment-backend/internal/models"
	"apartment-backend/internal/repository"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type PackageHandler struct {
	packageRepo *repository.PackageRepository
}

func NewPackageHandler(packageRepo *repository.PackageRepository) *PackageHandler {
	return &PackageHandler{packageRepo: packageRepo}
}

func (h *PackageHandler) CreatePackage(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}
	userID := middleware.GetUserID(c)

	var req models.CreatePackageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	pkg, err := h.packageRepo.Create(c.Context(), buildingID, userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(pkg, "Package registered"))
}

func (h *PackageHandler) GetPackages(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	status := c.Query("status", "all")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	packages, err := h.packageRepo.GetByBuilding(c.Context(), buildingID, status, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.JSON(models.SuccessResponse(packages, ""))
}

func (h *PackageHandler) PickUp(c *fiber.Ctx) error {
	pkgID, err := uuid.Parse(c.Params("pkgId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid package ID"))
	}
	userID := middleware.GetUserID(c)

	if err := h.packageRepo.MarkPickedUp(c.Context(), pkgID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.JSON(models.SuccessResponse(nil, "Package picked up"))
}

func (h *PackageHandler) Notify(c *fiber.Ctx) error {
	pkgID, err := uuid.Parse(c.Params("pkgId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid package ID"))
	}

	if err := h.packageRepo.NotifyRecipient(c.Context(), pkgID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.JSON(models.SuccessResponse(nil, "Recipient notified"))
}

func (h *PackageHandler) GetMyPackages(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	packages, err := h.packageRepo.GetMyPackages(c.Context(), userID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.JSON(models.SuccessResponse(packages, ""))
}
