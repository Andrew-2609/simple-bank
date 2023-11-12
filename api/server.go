package api

import (
	"fmt"

	db "github.com/Andrew-2609/simple-bank/db/sqlc"
	"github.com/Andrew-2609/simple-bank/token"
	"github.com/Andrew-2609/simple-bank/util"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type Server struct {
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
	router     *gin.Engine
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)

	if err != nil {
		return nil, fmt.Errorf("could not create token maker: %w", err)
	}

	server := &Server{config: config, store: store, tokenMaker: tokenMaker}
	router := gin.Default()
	router.SetTrustedProxies([]string{"127.0.0.1"})

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", validateCurrency)
	}

	router.POST("/users", server.createUser)

	router.POST("/accounts", server.createAccount)
	router.GET("/accounts", server.listAccounts)
	router.GET("/accounts/:id", server.getAccount)
	router.PUT("/accounts/:id", server.updateAccount)
	router.DELETE("/accounts/:id", server.deleteAccount)

	router.POST("/transfers", server.createTransfer)

	server.router = router

	return server, nil
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
