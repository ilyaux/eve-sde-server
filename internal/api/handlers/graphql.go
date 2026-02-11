package handlers

import (
	"database/sql"
	"net/http"

	"github.com/graphql-go/handler"
	customgraphql "github.com/ilya/eve-sde-server/internal/graphql"
	"github.com/rs/zerolog/log"
)

type GraphQLHandler struct {
	handler *handler.Handler
}

// NewGraphQLHandler creates a new GraphQL handler
func NewGraphQLHandler(db *sql.DB) (*GraphQLHandler, error) {
	schema, err := customgraphql.BuildSchema(db)
	if err != nil {
		return nil, err
	}

	h := handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true, // Enable GraphiQL UI
	})

	log.Info().Msg("GraphQL schema initialized")

	return &GraphQLHandler{handler: h}, nil
}

// ServeHTTP handles GraphQL requests
func (h *GraphQLHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}
