WebAssembly tools

WebAssembly spec https://webassembly.github.io/spec/core/

* Run a .wast file

```
$ go run ./cmd/run file.wast
```

* Convert a .wasm file into .wat

```
$ go run ./cmd/run file.wasm
```

* Run tests

```
$ go run ./test
```

So far this implementation of WebAssembly is ~3% complete, but it can instantiate a primitive webassembly module and run some basic instructions.
