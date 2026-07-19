package handlers

import (
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/httpsms/pkg/validators"
	"github.com/NdoleStudio/stacktrace"
	"github.com/davecgh/go-spew/spew"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// ContactHandler handles contact http requests.
type ContactHandler struct {
	handler
	logger    telemetry.Logger
	tracer    telemetry.Tracer
	validator *validators.ContactHandlerValidator
	service   *services.ContactService
}

// NewContactHandler creates a new ContactHandler.
func NewContactHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	validator *validators.ContactHandlerValidator,
	service *services.ContactService,
) (h *ContactHandler) {
	return &ContactHandler{
		logger:    logger.WithService(fmt.Sprintf("%T", h)),
		tracer:    tracer,
		validator: validator,
		service:   service,
	}
}

// RegisterRoutes registers the routes for the ContactHandler.
func (h *ContactHandler) RegisterRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	h.register(router, fiber.MethodGet, "/v1/contacts", middlewares, h.Index)
	h.register(router, fiber.MethodPost, "/v1/contacts", middlewares, h.Store)
	h.register(router, fiber.MethodPost, "/v1/contacts/upload", middlewares, h.Upload)
	h.register(router, fiber.MethodPut, "/v1/contacts/:contactID", middlewares, h.Update)
	h.register(router, fiber.MethodDelete, "/v1/contacts/:contactID", middlewares, h.Delete)
}

// Index lists contacts for the authenticated user.
// @Summary      List contacts
// @Description  Returns the paginated list of contacts for the authenticated user. The top-level "total" field is the number of contacts matching the query filter, independent of skip/limit, so clients can drive server-side pagination.
// @Security	 ApiKeyAuth
// @Tags         Contacts
// @Accept       json
// @Produce      json
// @Param        skip	query  int  	false	"number of contacts to skip"	minimum(0)
// @Param        query	query  string  	false 	"filter contacts containing query"
// @Param        limit	query  int  	false	"number of contacts to return"	minimum(1)	maximum(100)
// @Success      200 	{object}	responses.ContactsResponse
// @Failure      400	{object}	responses.BadRequest
// @Failure 	 401    {object}	responses.Unauthorized
// @Failure      422	{object}	responses.UnprocessableEntity
// @Failure      500	{object}	responses.InternalServerError
// @Router       /contacts [get]
func (h *ContactHandler) Index(c fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	var request requests.ContactIndex
	if err := c.Bind().Query(&request); err != nil {
		ctxLogger.Warn(stacktrace.Propagatef(err, "cannot marshall params [%s] into %T", c.OriginalURL(), request))
		return h.responseBadRequest(c, err)
	}

	sanitized := request.Sanitize()
	if errors := h.validator.ValidateIndex(ctx, sanitized); len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewErrorf("validation errors [%s], while listing contacts [%+#v]", spew.Sdump(errors), sanitized))
		return h.responseUnprocessableEntity(c, errors, "validation errors while listing contacts")
	}

	userID := h.userIDFomContext(c)
	params := sanitized.ToIndexParams()
	contacts, err := h.service.Index(ctx, userID, params)
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot list contacts for user [%s]", userID))
		return h.responseInternalServerError(c)
	}

	total, err := h.service.Count(ctx, userID, params)
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot count contacts for user [%s]", userID))
		return h.responseInternalServerError(c)
	}

	return h.responseOKWithTotal(c, fmt.Sprintf("fetched %d %s", len(*contacts), h.pluralize("contact", len(*contacts))), contacts, total)
}

// Store creates one or many contacts.
// @Summary      Create one or many contacts
// @Description  Creates a single contact or a batch of contacts. Accepts a JSON array or an object with a "contacts" array.
// @Security	 ApiKeyAuth
// @Tags         Contacts
// @Accept       json
// @Produce      json
// @Param        payload   body 		requests.ContactStoreRequest 	true 	"Contact(s) to create"
// @Success      201 	{object}	responses.ContactsResponse
// @Failure      400	{object}	responses.BadRequest
// @Failure 	 401    {object}	responses.Unauthorized
// @Failure      422	{object}	responses.UnprocessableEntity
// @Failure      500	{object}	responses.InternalServerError
// @Router       /contacts [post]
func (h *ContactHandler) Store(c fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	var request requests.ContactStoreRequest
	if err := c.Bind().Body(&request); err != nil {
		ctxLogger.Warn(stacktrace.Propagatef(err, "cannot marshall body [%s] into %T", c.Body(), request))
		return h.responseBadRequest(c, err)
	}

	sanitized := request.Sanitize()
	if errors := h.validator.ValidateStore(ctx, sanitized); len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewErrorf("validation errors [%s], while creating contacts", spew.Sdump(errors)))
		return h.responseUnprocessableEntity(c, errors, "validation errors while creating contacts")
	}

	userID := h.userIDFomContext(c)
	contacts := sanitized.ToContacts(userID)
	if err := h.service.CreateMany(ctx, userID, contacts); err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot create [%d] contacts for user [%s]", len(contacts), userID))
		return h.responseInternalServerError(c)
	}

	return h.responseCreated(c, fmt.Sprintf("created %d %s", len(contacts), h.pluralize("contact", len(contacts))), contacts)
}

// Upload imports contacts from a CSV file.
// @Summary      Import contacts from CSV
// @Description  Uploads a CSV file (multipart field "document") of contacts. Columns: Name, Emails, PhoneNumbers (multi-values separated by ";").
// @Security	 ApiKeyAuth
// @Tags         Contacts
// @Accept       multipart/form-data
// @Produce      json
// @Param        document	formData	file	true	"CSV file of contacts"
// @Success      201 	{object}	responses.ContactsResponse
// @Failure      400	{object}	responses.BadRequest
// @Failure 	 401    {object}	responses.Unauthorized
// @Failure      422	{object}	responses.UnprocessableEntity
// @Failure      500	{object}	responses.InternalServerError
// @Router       /contacts/upload [post]
func (h *ContactHandler) Upload(c fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	file, err := c.FormFile("document")
	if err != nil {
		ctxLogger.Warn(stacktrace.Propagatef(err, "cannot fetch file with name [%s] from request", "document"))
		return h.responseBadRequest(c, err)
	}

	userID := h.userIDFomContext(c)
	items, errors := h.validator.ValidateUpload(ctx, userID, file)
	if len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewErrorf("validation errors [%s], while importing contacts from CSV [%s]", spew.Sdump(errors), file.Filename))
		return h.responseUnprocessableEntity(c, errors, "validation errors while importing contacts")
	}

	// items are already sanitized by ValidateUpload (SanitizeContactItem), so
	// build the persistable records directly without re-sanitizing.
	request := requests.ContactStoreRequest{Contacts: items}
	contacts := request.ToContacts(userID)
	if err = h.service.CreateMany(ctx, userID, contacts); err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot import [%d] contacts for user [%s]", len(contacts), userID))
		return h.responseInternalServerError(c)
	}

	return h.responseCreated(c, fmt.Sprintf("imported %d %s", len(contacts), h.pluralize("contact", len(contacts))), contacts)
}

// Update updates a single contact.
// @Summary      Update a contact
// @Description  Updates the details of a single contact.
// @Security	 ApiKeyAuth
// @Tags         Contacts
// @Accept       json
// @Produce      json
// @Param 		 contactID	path		string 							true 	"ID of the contact"
// @Param        payload   	body 		requests.ContactUpdateRequest 	true 	"Contact details to update"
// @Success      200 		{object}	responses.ContactResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      404		{object}	responses.NotFound
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /contacts/{contactID} [put]
func (h *ContactHandler) Update(c fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	contactID := c.Params("contactID")
	if errors := h.validator.ValidateUUID(contactID, "contactID"); len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewErrorf("validation errors [%s], while updating contact [%s]", spew.Sdump(errors), contactID))
		return h.responseUnprocessableEntity(c, errors, "validation errors while updating contact")
	}

	var request requests.ContactUpdateRequest
	if err := c.Bind().Body(&request); err != nil {
		ctxLogger.Warn(stacktrace.Propagatef(err, "cannot marshall body into %T", request))
		return h.responseBadRequest(c, err)
	}

	sanitized := request.Sanitize()
	if errors := h.validator.ValidateUpdate(ctx, sanitized); len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewErrorf("validation errors [%s], while updating contact [%s]", spew.Sdump(errors), contactID))
		return h.responseUnprocessableEntity(c, errors, "validation errors while updating contact")
	}

	userID := h.userIDFomContext(c)
	contact, err := h.service.Get(ctx, userID, uuid.MustParse(contactID))
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return h.responseNotFound(c, fmt.Sprintf("cannot find contact with ID [%s]", contactID))
	}
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot load contact [%s] for user [%s]", contactID, userID))
		return h.responseInternalServerError(c)
	}

	sanitized.ApplyTo(contact)
	if err = h.service.Update(ctx, contact); err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot update contact [%s] for user [%s]", contactID, userID))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "contact updated successfully", contact)
}

// Delete removes a single contact.
// @Summary      Delete a contact
// @Description  Deletes a single contact from the database.
// @Security	 ApiKeyAuth
// @Tags         Contacts
// @Accept       json
// @Produce      json
// @Param 		 contactID	path		string 	true	"ID of the contact"
// @Success      204  	{object} 	responses.NoContent
// @Failure      400  	{object}  	responses.BadRequest
// @Failure 	 401    {object}	responses.Unauthorized
// @Failure 	 404	{object}	responses.NotFound
// @Failure      422  	{object} 	responses.UnprocessableEntity
// @Failure      500  	{object}  	responses.InternalServerError
// @Router       /contacts/{contactID} [delete]
func (h *ContactHandler) Delete(c fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	contactID := c.Params("contactID")
	if errors := h.validator.ValidateUUID(contactID, "contactID"); len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewErrorf("validation errors [%s], while deleting contact [%s]", spew.Sdump(errors), contactID))
		return h.responseUnprocessableEntity(c, errors, "validation errors while deleting contact")
	}

	userID := h.userIDFomContext(c)

	// Load first so a missing contact for the authenticated user returns 404
	// instead of silently succeeding via a 0-row DELETE. This also prevents
	// leaking whether another user's contact exists.
	if _, err := h.service.Get(ctx, userID, uuid.MustParse(contactID)); err != nil {
		if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
			return h.responseNotFound(c, fmt.Sprintf("cannot find contact with ID [%s]", contactID))
		}
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot load contact [%s] for user [%s]", contactID, userID))
		return h.responseInternalServerError(c)
	}

	if err := h.service.Delete(ctx, userID, uuid.MustParse(contactID)); err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot delete contact [%s] for user [%s]", contactID, userID))
		return h.responseInternalServerError(c)
	}

	return h.responseNoContent(c, "contact deleted successfully")
}
