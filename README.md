# goas
> Based on [yvasiyarov/swagger](https://github.com/yvasiyarov/swagger) repository.

Generate [OpenAPI Specification](https://swagger.io/specification) json file with comments in Go.

## Limit
Currently, only JSON supported

## Install

```
go get -u github.com/mikunalpha/goas/cmd/goas
```

## Usage
Comments in main.go
```go
// @Version 1.0.0
// @Title Backend API
// @Description API usually works as expected. But sometimes its not true.
// @ContactName Abcd
// @ContactEmail abce@email.com
// @ContactURL http://someurl.oxox
// @TermsOfServiceUrl http://someurl.oxox
// @LicenseName MIT
// @LicenseURL https://en.wikipedia.org/wiki/MIT_License
```
Comments for API handleFunc
```go
type User struct {
  Id   uint64 `json:"id"`
  Name string `json:"name"`
}

type UsersResponse struct {
  Data []Users `json:"users"`
}

type Error struct {
  Code string `json:"code"`
  Msg  string `json:"msg"`
}

type ErrorResponse struct {
  ErrorInfo Error `json:"error"`
}

// @Title Get user list of a group.
// @Description Get users related to a specific group.
// @Param  group_id  path  int  true  "Id of a specific group."
// @Success  200  {object}  UsersResponse  "UsersResponse JSON"
// @Failure  400  {object}  ErrorResponse  "ErrorResponse JSON"
// @Resource users
// @Router /api/group/{group_id}/users [get]
func GetGroupUsers() {
  // ...
}

// @Title Get user list of a group.
// @Description Create a new user.
// @Param  user  body  User  true  "Info of a user."
// @Success  200  {object}  User           "UsersResponse JSON"
// @Failure  400  {object}  ErrorResponse  "ErrorResponse JSON"
// @Resource users
// @Router /api/user [post]
func PostUser() {
  // ...
}
```
Go to the folder where is main.go in
```
goas --output oas.json
```
