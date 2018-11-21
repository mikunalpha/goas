// @Version v0.1.0
// @Title Foo API
// @Description API usually works as expected. But sometimes its not true.
// @ContactName mikun
// @ContactEmail mikun800527@gmail.com
// @TermsOfServiceUrl http://google.com/
// @LicenseName MIT
// @LicenseUrl http://unknown
// @Server http://www.fake.com Server-1
// @Server http://www.fake2.com Server-2
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/mikunalpha/goas/example/handler"
)

func main() {
	r := gin.New()
	r.GET("/apis/v1/empty", handler.GetEmpty)
	r.Run(":8080")
}
