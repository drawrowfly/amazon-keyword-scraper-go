## My First App In Go

# Amazon Keyword Suggestion Tool
From 1 keyword you can get up to hundreds and even thousands unique and relevant keywords ready to be used on Amazon

Result will be saved to a csv file

```go
go run main.go -keyword "iphone case" -limit 200
```

## Result in CLI
```
Amazon KeyWord Collector Started. Collect 900 relevant keywords for the keyword 'iphone case' 
Result: 10 keywords related to the keyword 'iphone case' were saved to the 'iphone case.csv' file 
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