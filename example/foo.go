package main

import (
	"encoding/json"
	"time"
)

type FooResponse struct {
	ID        string          `json:"id"`
	StartDate time.Time       `json:"startDate"`
	Msg       json.RawMessage `json:"msg"`
	InnerFoos []InnerFoo      `json:"foo"`
	Environments map[string]Environment `json:"environments"`
}

type Environment struct {
	Name string `json:"name"`
}

// @Title Get all foos
// @Description Get all foos
// @Route /api/v2/foo [get]
// @Success 200 object FooResponse "Successful foo response"
// @Failure 401 "Invalid access token"
// @Failure 403 "Forbidden"
// @Failure 404 "Invalid resource identifier"
func getAllFoos() {

}

// @Title Put foo
// @Description Overwrite a foo
// @Route /api/v2/foo [put]
// @Success 200 object FooResponse "Successful foo response"
// @Failure 401 "Invalid access token"
// @Failure 403 "Forbidden"
// @Failure 404 "Invalid resource identifier"
func putFoo() {

}

