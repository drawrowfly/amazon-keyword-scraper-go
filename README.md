## My First App In Go

# Amazon Keyword Suggestion Tool
From 1 keyword you can get up to hundreds and even thousands unique and relevant keywords ready to be used on Amazon

```go
go run main.go -keyword "iphone case" -limit 200
```

# Commands
```
  -keyword string
        keyword to use (default "iphone")
  -limit int
        number of keywords to collect (default 100)
  -concurency int
        the number of goroutines that are allowed to run concurrently (default 2)
```