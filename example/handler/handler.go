package handler

import "github.com/gin-gonic/gin"

type EmptyResp struct{}

type ErrResp struct {
	Error Err `json:"error"`
}

type Err struct {
	Code    string `json:"code"`
	Message string `json:"msg"`
}

type CatsResp struct {
	Data []Cat `json:"data"`
}

type CatReq struct {
	Data *Cat `json:"data"`
}

type CatResp struct {
	Data Cat `json:"data"`
}

type Cat struct {
	Name   *string         `json:"name,omitempty"`
	Age    int64           `json:"age,omitempty"`
	Phones *[]string       `json:"phones,omitempty"`
	Inter  map[string]bool `json:"inter,omitempty"`
	// Inter interface{} `json:"inter,omitempty"`
}

// GetEmpty returns empty object
// @Title Get Empty
// @Description Return empty object
// @Success  200  {object}  EmptyResp  "Empty"
// @Failure  500  {object}  ErrResp    "Error"
// @Resource empty
// @Router /apis/v1/empty [get]

func GetEmpty(c *gin.Context) {

}

// GetCat returns cats
// @Title Get Cats
// @Description Return cats
// @Success  200  {object}  CatsResp  "Cats"
// @Failure  500  {object}  ErrResp   "Error"
// @Resource cat
// @Router /apis/v1/cats [get]
func GetCats(c *gin.Context) {

}

// GetCat returns cat object
// @Title Get Cat
// @Description Return cat object
// @Param  id  path  int32  true  "Cat ID"
// @Success  200  {object}  CatResp  "Cat"
// @Failure  500  {object}  ErrResp  "Error"
// @Resource cat
// @Router /apis/v1/cats/{id} [get]
func GetCat(c *gin.Context) {

}

// PostCat creates cat object
// @Title Post Cat
// @Description Create cat object
// @Param  cat  body  CatReq  true  "Cat"
// @Success  200  {object}  CatResp  "Cat"
// @Failure  500  {object}  ErrResp  "Error"
// @Resource cat
// @Router /apis/v1/cat [post]
func PostCat(c *gin.Context) {

}
