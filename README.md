## My First App In Go

# Amazon Keyword Suggestion Tool
From 1 keyword you can get up to hundred or even thousands **Unique and Relevant Keywords** with a **Number of Active Products** per each keyword and ready to be used on Amazon.

KeyWords are collected from Amazon Web API.
Result will be saved to a csv file.

#### Latest Release
[Download Latest Release](https://github.com/drawrowfly/amazon-keyword-suggestion-golang/releases/)

## Example 
```go
akst -keyword "iphone" -limit 300
```
## Result in CLI
![Demo](https://i.imgur.com/O2Dgehi.png)

## CSV Example
![Demo](https://i.imgur.com/OwCLSev.png)


# Commands
```
  -keyword string
        keyword to use (default "iphone")
  -limit int
        number of keywords to collect (default 100)
  -concurency int
        the number of goroutines that are allowed to run concurrently (default 2)
```


<a href="https://www.buymeacoffee.com/Usom2qC" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/default-blue.png" alt="Buy Me A Coffee" style="height: 41px !important;width: 174px !important;box-shadow: 0px 3px 2px 0px rgba(190, 190, 190, 0.5) !important;-webkit-box-shadow: 0px 3px 2px 0px rgba(190, 190, 190, 0.5) !important;" ></a>
