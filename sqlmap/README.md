# sqlmap

[![expr](https://img.shields.io/badge/go-reference-blue.svg?style=flat)](https://pkg.go.dev/github.com/muktihari/x/sqlmap)

sqlmap, stand for SQL Mapper, is a sql library to map *sql.Rows into Golang data type. 

It use `json` for encoding/decoding.

## Usage

```go
...
import "github.com/muktihari/x/sqlmap"
...

ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
defer cancel()

db, _ := sql.Open("postgres", "<connInfo>")
stmt, _ := db.Prepare("SELECT * FROM users WHERE city = $1")
sqlRows, _ := stmt.QueryContext(ctx, "Surabaya")

type User struct {
    ID        int    `json:"id"`
    FirstName string `json:"first_name"`
    LastName  string `json:"last_name"`
    City      string `json:"city"`
}

var users []User
if err := sqlmap.All(sqlRows, &users); err != nil {
    return err
}
```

### Handle `JSONB` data type:
For example we have `Product` and `Order` table and `Order` has column `product` type `JSONB` which is the screenshot of the purchased `Product` represented as json.
```go
...
import "github.com/muktihari/x/sqlmap/opt"
...

type Product struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}
type Order struct {
    ID        int       `json:"id"`
    Product   Product   `json:"product"`
    Qty       int       `json:"qty"`
    CreatedAt time.Time `json:"created_at"`
    UserID    int       `json:"user_id"`
}

stmt, _ := db.Prepare("SELECT * FROM order WHERE user_id = $1")
sqlRows, _ := stmt.QueryContext(ctx, 1)

var orders []Order
if err := sqlmap.All(sqlRows, &orders, opt.HandleJSONB); err != nil {
    return err
}
```

You can create your own option functions that satisfy `opt.Option` to handle the data the way you wanted. See `opt/opt.go` for example. 


