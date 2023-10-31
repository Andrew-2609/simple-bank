package api

import (
	"database/sql"
	"fmt"
	"net/http"

	db "github.com/Andrew-2609/simple-bank/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

type createAccountRequest struct {
	Owner    string `json:"owner" binding:"required"`
	Currency string `json:"currency" binding:"required,currency"`
}

func (server *Server) createAccount(ctx *gin.Context) {
	var req createAccountRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.CreateAccountParams{
		Owner:    req.Owner,
		Balance:  0,
		Currency: req.Currency,
	}

	newAccount, err := server.store.CreateAccount(ctx, arg)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "foreign_key_violation", "unique_violation":
				ctx.JSON(http.StatusUnprocessableEntity, errorResponse(err))
				return
			}
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, newAccount)
}

type listAccountsRequest struct {
	Page     int32 `form:"page" binding:"required,min=1"`
	Quantity int32 `form:"quantity" binding:"max=200"`
}

func (server *Server) listAccounts(ctx *gin.Context) {
	var req listAccountsRequest

	if err := ctx.BindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if req.Quantity == 0 {
		req.Quantity = 40
	}

	foundAccounts, err := server.store.ListAccounts(ctx, db.ListAccountsParams{
		Limit:  req.Quantity,
		Offset: (req.Page - 1) * req.Quantity,
	})

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.Header("total", fmt.Sprint(len(foundAccounts)))
	ctx.JSON(http.StatusOK, foundAccounts)
}

type getAccountRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) getAccount(ctx *gin.Context) {
	var req getAccountRequest

	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	foundAccount, err := server.store.GetAccount(ctx, req.ID)

	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, foundAccount)
}

type updateAccountRequest struct {
	params struct {
		ID int64 `uri:"id" binding:"required,min=1"`
	}
	body struct {
		Balance int64 `json:"balance" binding:"required,min=1"`
	}
}

func (server *Server) updateAccount(ctx *gin.Context) {
	var req updateAccountRequest

	if err := ctx.ShouldBindUri(&req.params); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if err := ctx.ShouldBindJSON(&req.body); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	updatedAccount, err := server.store.UpdateAccount(ctx, db.UpdateAccountParams{
		ID:      req.params.ID,
		Balance: req.body.Balance,
	})

	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, updatedAccount)
}

type deleteAccountRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) deleteAccount(ctx *gin.Context) {
	var req deleteAccountRequest

	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if err := server.store.DeleteAccount(ctx, req.ID); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
