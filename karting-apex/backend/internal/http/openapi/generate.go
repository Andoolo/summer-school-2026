package openapi

//go:generate npm --prefix ../../../../01-analysis/api run bundle
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.0 -config config/auth.yaml ../../../../01-analysis/api/dist/auth.yaml
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.0 -config config/slots.yaml ../../../../01-analysis/api/dist/slots.yaml
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.0 -config config/bookings.yaml ../../../../01-analysis/api/dist/bookings.yaml
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.0 -config config/profile.yaml ../../../../01-analysis/api/dist/profile.yaml
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.0 -config config/instructors.yaml ../../../../01-analysis/api/dist/instructors.yaml
