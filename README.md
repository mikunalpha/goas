# goas
The project is based on  
- [yvasiyarov/swagger](https://github.com/yvasiyarov/swagger) repository.  
- [uudashr/go-module](https://github.com/uudashr/go-module) repository.

Generate [OpenAPI Specification](https://swagger.io/specification) json file with comments in Go.

## Limit
- Only support go module.
- Anonymous struct field is not supported.

## Install

```
go get -u github.com/mikunalpha/goas
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
// @Server http://www.fake.com Server-1
// @Server http://www.fake2.com Server-2
```
Comments for API handleFunc
```go
type User struct {
  ID   uint64 `json:"id" example:"100" description:"User identity"`
  Name string `json:"name" example:"Mikun"` 
}

type UsersResponse struct {
  Data []Users `json:"users" example:"[{\"id\":100, \"name\":\"Mikun\"}]"`
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
// @Param  groupID  path  int  true  "Id of a specific group."
// @Success  200  object  UsersResponse  "UsersResponse JSON"
// @Failure  400  object  ErrorResponse  "ErrorResponse JSON"
// @Resource users
// @Route /api/group/{groupID}/users [get]
func GetGroupUsers() {
  // ...
}

// @Title Get user list of a group.
// @Description Create a new user.
// @Param  user  body  User  true  "Info of a user."
// @Success  200  object  User           "UsersResponse JSON"
// @Failure  400  object  ErrorResponse  "ErrorResponse JSON"
// @Resource users
// @Route /api/user [post]
func PostUser() {
  // ...
}
```
Go to the folder where is main.go in
```
// go.mod and main file are in the same directory
goas --module-path . --output oas.json

// go.mod and main file are in the different directory
goas --module-path . --main-file-path ./cmd/xxx/main.go --output oas.json
```

## HandleFunc Comments

### Title & Description
```
@Title {title}
@Title Get user list of a group.

@Description {description}.
@Description Get users related to a specific group.
```
- {title}: The title of the route.
- {description}: The description of the route.

### Parameter
```
@Param  {name}  {in}  {goType}  {required}  {description}
@Param  user    body  User      true        "Info of a user."
```
- {name}: The parameter name.
- {in}: The parameter is in `path`, `query`, `form`, `header`, `cookie`, `body` or `file`.
- {goType}: The type in go code. This will be ignored when {in} is `file`.
- {required}: `true`, `false`, `required` or `optional`. 
- {description}: The description of the parameter. Must be quoted.

### Response
```
@Success  {stauts}  {jsonType}  {goType}       {description}
@Success  200       object      UsersResponse  "UsersResponse JSON"

@Failure  {stauts}  {jsonType}  {goType}       {description}
@Failure  400       object      ErrorResponse  "ErrorResponse JSON"
```
- {status}: The HTTP status code.
- {jsonType}: The value can be `object` or `array`. 
- {goType}: The type in go code.
- {description}: The description of the response. Must be quoted.

### Resource & Tag
```
@Resource {resource}
@Resource users

@Tag {tag}
@tag xxx
```
- {resource}, {tag}: Tag of the route.

### Route
```
@Route {path}    {method}
@Route /api/user [post]
```
- {path}: The URL path.
- {method}: The HTTP Method. Must be put in brackets.