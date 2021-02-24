package main

// @Title Get inner foos
// @Description Get Inner Foos
// @Route /api/v2/foo/{id}/inner [put]
// @Success 200 object InnerFoo "Successful innerfoo response"
// @Failure 401 "Invalid access token"
// @Failure 403 "Forbidden"
// @Failure 404 "Invalid resource identifier"
func getInnerFoo() {

}

type InnerFoo struct {
	A string `json:"a"`
	B string `json:"b"`
}
