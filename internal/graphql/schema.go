package graphql

import (
	"database/sql"

	"github.com/graphql-go/graphql"
)

// BuildSchema creates the GraphQL schema
func BuildSchema(db *sql.DB) (graphql.Schema, error) {
	// Define Item type
	itemType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Item",
		Fields: graphql.Fields{
			"typeId": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
			},
			"name": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
			},
			"description": &graphql.Field{
				Type: graphql.String,
			},
			"volume": &graphql.Field{
				Type: graphql.Float,
			},
			"groupId": &graphql.Field{
				Type: graphql.Int,
			},
			"categoryId": &graphql.Field{
				Type: graphql.Int,
			},
			"published": &graphql.Field{
				Type: graphql.Boolean,
			},
		},
	})

	categoryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Category",
		Fields: graphql.Fields{
			"categoryId": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
			},
			"name": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
			},
			"published": &graphql.Field{
				Type: graphql.Boolean,
			},
		},
	})

	groupType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Group",
		Fields: graphql.Fields{
			"groupId": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
			},
			"categoryId": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
			},
			"name": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
			},
			"published": &graphql.Field{
				Type: graphql.Boolean,
			},
		},
	})

	// Define root Query
	rootQuery := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"item": &graphql.Field{
				Type: itemType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return resolveItem(db, p)
				},
			},
			"items": &graphql.Field{
				Type: graphql.NewList(itemType),
				Args: graphql.FieldConfigArgument{
					"limit": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 50,
					},
					"offset": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 0,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return resolveItems(db, p)
				},
			},
			"search": &graphql.Field{
				Type: graphql.NewList(itemType),
				Args: graphql.FieldConfigArgument{
					"query": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"limit": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 50,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return resolveSearch(db, p)
				},
			},
			"category": &graphql.Field{
				Type: categoryType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return resolveCategory(db, p)
				},
			},
			"categories": &graphql.Field{
				Type: graphql.NewList(categoryType),
				Args: graphql.FieldConfigArgument{
					"limit": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 50,
					},
					"offset": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 0,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return resolveCategories(db, p)
				},
			},
			"group": &graphql.Field{
				Type: groupType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return resolveGroup(db, p)
				},
			},
			"groups": &graphql.Field{
				Type: graphql.NewList(groupType),
				Args: graphql.FieldConfigArgument{
					"categoryId": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
					"limit": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 50,
					},
					"offset": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 0,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return resolveGroups(db, p)
				},
			},
		},
	})

	// Build schema
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: rootQuery,
	})

	return schema, err
}
