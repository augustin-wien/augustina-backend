package handlers

import (
	"errors"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
)

type voucherCodeRequest struct {
	Code string
}

// VoucherCheckResponse is what the redeem page shows before a voucher is redeemed
type VoucherCheckResponse struct {
	Code           string
	Status         string
	ItemName       string
	DurationMonths int
	RecipientName  string
	Message        string
}

type redeemVoucherRequest struct {
	Code      string
	Email     string
	FirstName string
	LastName  string
}

// RedeemVoucherResponse describes the abonement created by redeeming a voucher
type RedeemVoucherResponse struct {
	ItemName       string
	DurationMonths int
	FromDate       time.Time
	ToDate         time.Time
	Email          string
}

type resendVoucherRequest struct {
	Email string
}

func voucherErrorStatus(err error) int {
	switch {
	case errors.Is(err, database.ErrVoucherNotFound):
		return http.StatusNotFound
	case errors.Is(err, database.ErrVoucherRedeemed), errors.Is(err, database.ErrVoucherDisabled):
		return http.StatusConflict
	}
	return http.StatusBadRequest
}

// CheckVoucher godoc
//
//	@Summary		Check an abonement voucher
//	@Description	Returns the status of a voucher so the redeem page can show what it is for
//	@Tags			Vouchers
//	@Accept			json
//	@Produce		json
//	@Param			data body voucherCodeRequest true "Voucher code"
//	@Success		200	{object}	VoucherCheckResponse
//	@Router			/vouchers/check/ [post]
func CheckVoucher(w http.ResponseWriter, r *http.Request) {
	var req voucherCodeRequest
	if err := utils.ReadJSON(w, r, &req); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	v, err := database.Db.GetVoucherByCode(req.Code)
	if err != nil {
		if !errors.Is(err, database.ErrVoucherNotFound) {
			log.Error("CheckVoucher: ", err)
		}
		utils.ErrorJSON(w, database.ErrVoucherNotFound, voucherErrorStatus(database.ErrVoucherNotFound))
		return
	}
	respond(w, nil, VoucherCheckResponse{
		Code:           v.Code,
		Status:         v.Status,
		ItemName:       v.ItemName,
		DurationMonths: v.DurationMonths,
		RecipientName:  v.RecipientName,
		Message:        v.Message,
	})
}

// RedeemVoucher godoc
//
//	@Summary		Redeem an abonement voucher
//	@Description	Creates an online abonement for the email. A Keycloak account is created if needed; login details are only sent by mail.
//	@Tags			Vouchers
//	@Accept			json
//	@Produce		json
//	@Param			data body redeemVoucherRequest true "Code and email"
//	@Success		200	{object}	RedeemVoucherResponse
//	@Router			/vouchers/redeem/ [post]
func RedeemVoucher(w http.ResponseWriter, r *http.Request) {
	var req redeemVoucherRequest
	if err := utils.ReadJSON(w, r, &req); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(req.Email)
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		utils.ErrorJSON(w, errors.New("invalid email address"), http.StatusBadRequest)
		return
	}
	if len(req.FirstName) > 100 || len(req.LastName) > 100 {
		utils.ErrorJSON(w, errors.New("name is too long"), http.StatusBadRequest)
		return
	}
	result, err := database.Db.RedeemVoucher(database.RedeemVoucherRequest{
		Code:      req.Code,
		Email:     email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		status := voucherErrorStatus(err)
		if status == http.StatusBadRequest {
			log.Error("RedeemVoucher: ", err)
		}
		utils.ErrorJSON(w, err, status)
		return
	}
	respond(w, nil, RedeemVoucherResponse{
		ItemName:       result.ItemName,
		DurationMonths: result.DurationMonths,
		FromDate:       result.FromDate,
		ToDate:         result.ToDate,
		Email:          strings.ToLower(email),
	})
}

// ListVouchers godoc
//
//	@Summary		List abonement vouchers
//	@Tags			Vouchers
//	@Produce		json
//	@Success		200	{array}	database.AbonementVoucher
//	@Security		KeycloakAuth
//	@Router			/vouchers/ [get]
func ListVouchers(w http.ResponseWriter, r *http.Request) {
	vouchers, err := database.Db.ListVouchers()
	respond(w, err, vouchers)
}

func voucherIDParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, errors.New("invalid voucher id"), http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// GetVoucherPDF godoc
//
//	@Summary		Download the PDF of an abonement voucher
//	@Tags			Vouchers
//	@Produce		application/pdf
//	@Param			id path int true "Voucher ID"
//	@Security		KeycloakAuth
//	@Router			/vouchers/{id}/pdf/ [get]
func GetVoucherPDF(w http.ResponseWriter, r *http.Request) {
	id, ok := voucherIDParam(w, r)
	if !ok {
		return
	}
	v, err := database.Db.GetVoucherByID(id)
	if err != nil {
		utils.ErrorJSON(w, err, voucherErrorStatus(err))
		return
	}
	pdf, err := database.Db.VoucherPDF([]database.AbonementVoucher{v})
	if err != nil {
		log.Error("GetVoucherPDF: ", err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="Gutschein-`+v.Code+`.pdf"`)
	_, _ = w.Write(pdf)
}

// ResendVoucher godoc
//
//	@Summary		Send the PDF of an abonement voucher again
//	@Description	Sends it to the buyer, or to the given email
//	@Tags			Vouchers
//	@Accept			json
//	@Produce		json
//	@Param			id path int true "Voucher ID"
//	@Param			data body resendVoucherRequest false "Optional other email"
//	@Security		KeycloakAuth
//	@Router			/vouchers/{id}/resend/ [post]
func ResendVoucher(w http.ResponseWriter, r *http.Request) {
	id, ok := voucherIDParam(w, r)
	if !ok {
		return
	}
	var req resendVoucherRequest
	if r.ContentLength > 0 {
		if err := utils.ReadJSON(w, r, &req); err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
	}
	email := strings.TrimSpace(req.Email)
	if email != "" {
		if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
			utils.ErrorJSON(w, errors.New("invalid email address"), http.StatusBadRequest)
			return
		}
	}
	if err := database.Db.ResendVoucherMail(id, email); err != nil {
		log.Error("ResendVoucher: ", err)
		utils.ErrorJSON(w, err, voucherErrorStatus(err))
		return
	}
	respond(w, nil, map[string]bool{"sent": true})
}

func setVoucherDisabled(w http.ResponseWriter, r *http.Request, disabled bool) {
	id, ok := voucherIDParam(w, r)
	if !ok {
		return
	}
	v, err := database.Db.SetVoucherDisabled(id, disabled)
	if err != nil {
		utils.ErrorJSON(w, err, voucherErrorStatus(err))
		return
	}
	respond(w, nil, v)
}

// DisableVoucher godoc
//
//	@Summary		Disable an abonement voucher so it can't be redeemed
//	@Tags			Vouchers
//	@Produce		json
//	@Param			id path int true "Voucher ID"
//	@Success		200	{object}	database.AbonementVoucher
//	@Security		KeycloakAuth
//	@Router			/vouchers/{id}/disable/ [post]
func DisableVoucher(w http.ResponseWriter, r *http.Request) {
	setVoucherDisabled(w, r, true)
}

// EnableVoucher godoc
//
//	@Summary		Enable a disabled abonement voucher again
//	@Tags			Vouchers
//	@Produce		json
//	@Param			id path int true "Voucher ID"
//	@Success		200	{object}	database.AbonementVoucher
//	@Security		KeycloakAuth
//	@Router			/vouchers/{id}/enable/ [post]
func EnableVoucher(w http.ResponseWriter, r *http.Request) {
	setVoucherDisabled(w, r, false)
}
