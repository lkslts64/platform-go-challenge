
# GWI Go Challenge
---------

This is my solution for the [GWI Go Challenge](https://github.com/GlobalWebIndex/platform-go-challenge).


## Features

Aside from the required functionality, the following features where added:

* Authentication with JWT
* Rate limiting
* Pagination
* Metrics ([expvar](https://pkg.go.dev/expvar))
* Testing and Benchmark suite
* Docker

## Storage

For the storage layer, a thread-safe memory store was implemented. The store is optimized for read operations (read all users, all assets or a user's favourite assets). The storage is preloaded with a number of assets on service initialization.

One could ask why not use a persistent storage since the number of objects could become quite large and not fit in memory. In fact, I've assumed that the number of objects will not surpass a certain limit. Although I've made this assumption, great care has been taken into choosing the right data types to minimize each object's memory footprint.

For thread-safety, I chose to back up every `map` of the memory store with a `sync.RWMutex`. `sync.Map` was also a choice, but it is specialized for certain access patterns which shouldn't show up in this service.

## Web Server

For the web server, the standard's library `http.Server` is used along with gorilla's `mux` package for routing. The server is configured with sane defaults. Practices such as graceful shutdown, rate limiting, authentication, logging/recovery middleware and health check endpoints have been applied. Furthermore, a metrics endpoint is registered under `/metrics`.


## API Reference


### Assets

#### `GET /assets` 

Get all available assets.

#### `GET /asset/{id}`

Get information about a specific asset.

#### `POST /asset`

Create a new asset. Examples for the request body:


```json
{
  "type": "insight",
  "description": "awesome insight",
  "data": {
    "text": "40% of millenials spend more than 3hours on social media daily"
  }
}
```

```json
{
  "type": "chart",
  "description": "awesome chart",
  "data": {
    "title": "awesome chart",
    "titleAxisX": "axisX",
    "titleAxisY": "axisY",
    "data": "cmFuZG9tIGJhc2U2NCBzdHJpbmcK"
  }
}
```

```json
{
  "type": "audience",
  "description": "awesome audience",
  "data": {
    "gender": "male",
    "birthCountry": "Greece",
    "socialMediaHoursUsage": 0,
    "ageGroup": {
      "min": 15,
      "max": 30
    }
  }
}
```


#### `DELETE /asset/{id}`

Delete a specific asset. If the asset is present in the favourites of any user, it will be removed from the favourites too. 

#### `PUT /asset/{id}`

Update a specific asset. In the request body, specify only the fields of the asset that you are willing to update. For example, to update only the description of an asset:

```json
{
	"description" : "new description"
}
```


### Users

#### `GET /users` 

Get all available users.

#### `GET /user/{id}`

Get information about a specific user.

#### `POST /user`

Create a new user. Example request body:

```json
{
  "name": "Lucas Litsos",
  "email": "lkslts64@gmail.com"
}
```

#### `DELETE /user/{id}`

Delete a specific user along with all of its favourites.

#### `PUT /user/{id}`

Update a specific user. In the request body, specify only the fields of the user that you are willing to update. For example, to update only the name of an user:

```json
{
	"name" : "new name"
}
```

### Favourites

#### `GET /users/{id}/favourites` 

Get all favourites of a specific user. This endpoint supports two optional query parameters: `page` and `limit`.
`page` specifies the page of results to return. `limit` caps the number of results.

#### `PUT /users/{id}/favourites/{assetID}` 

Add an asset to a user's favourites.

#### `DELETE /users/{id}/favourites/{assetID}` 

Delete an asset from a user's favourites.


## Build/Run

If you want to run with Docker:

     docker build -t gwitha .
     docker  run -p  8080:8080 gwitha

or directly from source:

     go run main.go 


## Interacting with the API

*Note: the API testing CLI client used in the examples is [httpie](https://httpie.io/).*

First you have to login:

    http POST localhost:8080/login username=gwi password=gwi

In the set cookie, you will find a JWT token like this one:

    eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6Imd3aSIsImV4cCI6MTY1MjgwMTkwNn0.plr_BQJV8qxwby3Mx1GuLxK2ybyF9Z3yDa8Yo3hvLcM
  
In every subsequent request, you have to include the JWT token in the Authorization header:

    Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6Imd3aSIsImV4cCI6MTY1MjgwMTkwNn0.plr_BQJV8qxwby3Mx1GuLxK2ybyF9Z3yDa8Yo3hvLcM
  
You can now make requests to the API.

To get a list of all assets:

    http GET localhost:8080/assets

or assets with a specific type:
  
    http GET 'localhost:8080/assets?type=chart'

To get a list of all users:

    http GET localhost:8080/users


To get a list of a user's 1 favourite assets:

    http GET localhost:8080/users/1/favourites


The former endpoint supports pagination, limiting the number of results and specifying the type of assets to return.

    http GET 'localhost:8080/users/1/favourites?page=1&limit=100&type=insight'

This section does not cover all endpoints. See the API reference for more.


## Usage 

```
  -port string
        port to listen on (default "8080")
  -ratelimit
        enable rate limiting
```


## Testing

Under the project's root directory, run:

	 cd service && go test ./...



## Dependencies

* [github.com/gorilla/mux](github.com/gorilla/mux)
* [github.com/golang-jwt/jwt](github.com/golang-jwt/jwt)
* [github.com/gorilla/handlers](github.com/gorilla/handlers)
* [golang.org/x/time](golang.org/x/time)
* [github.com/stretchr/testify](github.com/stretchr/testify)