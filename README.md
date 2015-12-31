# nsqscript

Simple script interpreter, wrapping the nsq HTTP API.

![Progress](http://progressed.io/bar/30?title=Underway)

This package is the interpreter used as the backend by projects such as [nsqrepl](https://github.com/russmack/nsqrepl) and nsqscripter.

## Usage
```
import "github.com/russmack/nsqscript"
...
res := nsqscript.ParseLine("ping ip 127.0.0.1")
```
Or
```
import "github.com/russmack/nsqscript"
...
f, _ := os.Open(filename)
resultsChan := make(chan string)
go nsqscript.ParseScript(f, resultsChan)
for r := range resultsChan {
  fmt.Println(r)
}
```

## License
BSD 3-Clause: [LICENSE.txt](LICENSE.txt)

[<img alt="LICENSE" src="http://img.shields.io/pypi/l/Django.svg?style=flat-square"/>](LICENSE.txt)
