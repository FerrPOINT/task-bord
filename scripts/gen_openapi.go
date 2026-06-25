package main

import (
	"encoding/json"
	"fmt"

	apiv1 "github.com/FerrPOINT/task-bord/pkg/routes/api/v1"

	"github.com/labstack/echo/v5"
)

func main() {
	e := echo.New()
	g := e.Group("/api/v1")
	api := apiv1.NewAPI(e, g)
	apiv1.RegisterAll(api)
	b, err := json.MarshalIndent(api.OpenAPI(), "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}
