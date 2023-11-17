# Copier

This is a simple Struct and Slice copier written in golang.

## Install

```shell
go get github.com/PPG007/copier
```

## Usage

### Copy Struct

For example, there are two structs:

```go
type S1 struct {
	Id        string
}

type S2 struct {
	Id        string
}
```

Now we can copy value from S1 to S2:

```go
func main() {
	s1 := S1{Id: "id1"}
	s2 := S2{}
	err := copier.New(true).RegisterConverter(TimeStringConverter).From(s1).To(&s2)
}
```

The copier `New()` function receive a bool and determine should return error if two fields that have same name but different types.

### Copy Slice

Copier also support copy between slices, for example: 

```go
func main() {
    slice1 := []S1{
        {
            Id: "1",
        },
        {
            Id: "2",
        },
    }
    var slice2 []S2
    err := copier.New(true).From(slice1).To(&slice2)
}
```

### Custom type converter

Convert value from type a to type b is always useful, especially copy from a DB model to an HTTP response object, you may need convert some database type to string, e.g. MongoDB ObjectId.

You can register different type converters to resolve the problem, for example we want to trans time.Time to string in RFC3339:

```go
TimeStringConverter = Converter{
    Origin: reflect.TypeOf(time.Time{}),
    Target: reflect.TypeOf(""),
    Fn: func(fromValue reflect.Value, toType reflect.Type) (reflect.Value, error) {
        t, ok := fromValue.Interface().(time.Time)
        if ok {
            return reflect.ValueOf(t.Format(time.RFC3339)), nil
        }
        return fromValue, nil
    },
}
// this is already written in copier, just use it.
copier.New(true).RegisterConverter(copier.TimeStringConverter)
```

Now, the copier will convert time field to string in RFC3339.

### Use transformer to custom the copy process for one field

Sometimes we want to do more when a field is being copied, for example we don't want copy value simply:

```go
s1 := S1{
    Id:        "123",
    CreatedAt: time.Date(2023, time.February, 1, 0, 0, 0, 0, time.Local),
}
s2 := S2{}
err := copier.New(true).RegisterTransformer("Id", func(id string) int {
    n, _ := strconv.ParseInt(id, 10, 64)
    return int(n)
}).RegisterTransformer("CreatedAt", func(createdAt time.Time) string {
    return createdAt.Format("2006")
}).From(s1).To(&s2)
```

By using `RegisterTransformer()`, we can do the copy for one field ourselves.

### Copy to different name fields

If you want to copy a field from a to b, but their names are not same, you can use `RegisterDiffPairs()`:

```go
slice1 := []S1{
    {
        Id:        "1",
        CreatedAt: time.Now(),
    },
    {
        Id:        "2",
        CreatedAt: time.Now().AddDate(1, 0, 0),
    },
}
var slice2 []*S2
err := copier.New(true).RegisterConverter(TimeStringConverter).RegisterDiffPairs([]copier.DiffPair{
    {
        Origin: "Id",
        Target: []string{"Id2"},
    },
}).From(slice1).To(&slice2)
```

`RegisterDiffPairs()` can use together with `RegisterTransformer()`:

```go
copier.New(true).RegisterConverter(TimeStringConverter).RegisterDiffPairs([]copier.DiffPair{
    {
        Origin: "Id",
        Target: []string{"Id2"},
    },
}).RegisterTransformer("Id2", func(id string) int {
    n, _ := strconv.ParseInt(id, 10, 64)
    return int(n)
}).From(slice1).To(&slice2)
```

## Examples

You can find more examples in [copier_test.go](./copier_test.go).
