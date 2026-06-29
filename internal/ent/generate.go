package ent

//go:generate go mod download entgo.io/ent
//go:generate go run entgo.io/ent/cmd/ent@v0.14.5 generate --feature intercept ./schema
